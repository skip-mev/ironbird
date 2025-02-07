package testnet

import (
	"bytes"
	"fmt"
	"github.com/nao1215/markdown"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/observability"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/util"
	"github.com/skip-mev/petri/core/v3/monitoring"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

var testnetActivities *testnet.Activity
var githubActivities *github.NotifierActivity
var observabilityActivities *observability.Activity

func Workflow(ctx workflow.Context, opts WorkflowOptions) (string, error) {
	name := fmt.Sprintf("Testnet (%s) bake", opts.ChainConfig.Name)
	runName := fmt.Sprintf("ib-%s-%s", opts.ChainConfig.Name, opts.SHA[:6])
	start := workflow.Now(ctx)
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, options)
	output := ""

	checkId, err := createInitialCheck(ctx, opts, name)

	if err != nil {
		return "", err
	}

	builtTag, err := buildImage(ctx, opts)

	if err != nil {
		return "", err
	}

	testnetOptions := testnet.TestnetOptions{
		Name:       runName,
		Image:      builtTag,
		UID:        opts.ChainConfig.Image.UID,
		GID:        opts.ChainConfig.Image.GID,
		BinaryName: opts.ChainConfig.Image.BinaryName,
		HomeDir:    opts.ChainConfig.Image.HomeDir,
		ProviderSpecificOptions: map[string]string{
			"region":   "ams3",
			"image_id": "177032231",
			"size":     "s-1vcpu-1gb",
		},
		ValidatorCount: 4,
		NodeCount:      0,
	}

	var providerState string
	if err = workflow.ExecuteActivity(ctx, testnetActivities.CreateProvider, testnetOptions).Get(ctx, &providerState); err != nil {
		return "", err
	}

	testnetOptions.ProviderState = []byte(providerState)

	defer func() {
		newCtx, _ := workflow.NewDisconnectedContext(ctx)
		err := workflow.ExecuteActivity(newCtx, testnetActivities.TeardownProvider, testnetOptions).Get(newCtx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Error("failed to teardown provider", "error", err)
		}
	}()

	var chainState testnet.PackagedState

	if err = workflow.ExecuteActivity(ctx, testnetActivities.LaunchTestnet, testnetOptions).Get(ctx, &chainState); err != nil {
		return "", err
	}

	testnetOptions.ChainState = chainState.ChainState
	testnetOptions.ProviderState = chainState.ProviderState

	if err := appendNodeTable(chainState.Nodes, &output); err != nil {
		return "", err
	}

	var observabilityPackagedState observability.PackagedState
	var metricsIps []string

	for _, node := range chainState.Nodes {
		metricsIps = append(metricsIps, node.Metrics)
	}

	if err := workflow.ExecuteActivity(
		ctx,
		observabilityActivities.LaunchObservabilityStack,
		observability.Options{
			PrometheusTargets:      metricsIps,
			ProviderState:          testnetOptions.ProviderState,
			ProviderSpecificConfig: testnetOptions.ProviderSpecificOptions,
		},
	).Get(ctx, &observabilityPackagedState); err != nil {
		return "", err
	}

	testnetOptions.ProviderState = observabilityPackagedState.ProviderState

	output += fmt.Sprintf("## Observability\n- [Grafana](%s)\n", observabilityPackagedState.ExternalGrafanaURL)

	if err = monitorTestnet(ctx, opts, testnetOptions, checkId, name, &output, observabilityPackagedState.ExternalGrafanaURL); err != nil {
		return "", err
	}

	if err = updateCheck(ctx, checkId, opts.GenerateCheckOptions(
		name,
		"completed",
		"The testnet has successfully baked in",
		fmt.Sprintf("The bake in period took %s", workflow.Now(ctx).Sub(start).String()),
		output,
		util.StringPtr("success"),
	)); err != nil {
		return "", err
	}

	return "", err
}

func monitorTestnet(ctx workflow.Context, opts WorkflowOptions, testnetOptions testnet.TestnetOptions, checkId int64, name string, output *string, grafanaUrl string) error {
	for i := 0; i < 360; i++ {
		if err := workflow.Sleep(ctx, 10*time.Second); err != nil {
			return err
		}

		var status string
		// TODO: metrics checks
		err := workflow.ExecuteActivity(ctx, testnetActivities.MonitorTestnet, testnetOptions).Get(ctx, &status)

		if err != nil {
			return err
		}

		var screenshot []byte
		err = workflow.ExecuteActivity(ctx, observabilityActivities.GrabGraphScreenshot, grafanaUrl, monitoring.DefaultDashboardUID, "comet-performance", "18", "now-5m").Get(ctx, &screenshot)

		if err != nil {
			return err
		}

		var screenShotUrl string

		err = workflow.ExecuteActivity(ctx, observabilityActivities.UploadScreenshot, testnetOptions.Name, fmt.Sprintf("testnet-%d", time.Now().Unix()), screenshot).Get(ctx, &screenShotUrl)

		if err != nil {
			return err
		}

		*output += fmt.Sprintf("### Screenshot - %d\n ![](%s)\n", i, screenShotUrl)

		if err = updateCheck(ctx, checkId, opts.GenerateCheckOptions(
			name,
			"in_progress",
			fmt.Sprintf("Monitoring the testnet - %d", i),
			fmt.Sprintf("Monitoring the testnet - %d", i),
			*output,
			nil,
		)); err != nil {
			return err
		}

	}

	return nil
}

func appendNodeTable(nodes []testnet.Node, output *string) error {
	if output == nil {
		return fmt.Errorf("output cannot be nil")
	}

	var buf bytes.Buffer

	rows := [][]string{}

	for _, n := range nodes {
		rows = append(rows, []string{n.Name, n.Rpc, n.Lcd})
	}

	err := markdown.NewMarkdown(&buf).Table(markdown.TableSet{
		Header: []string{"Name", "RPC", "LCD"},
		Rows:   rows,
	}).Build()

	if err != nil {
		return nil
	}

	*output += fmt.Sprintf("## Nodes\n%s\n", buf.String())
	return nil
}
