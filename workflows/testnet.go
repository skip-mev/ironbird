package workflows

import (
	"bytes"
	"fmt"
	"github.com/nao1215/markdown"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/builder"
	"github.com/skip-mev/ironbird/types"
	"go.temporal.io/sdk/workflow"
	"os"
	"time"
)

var testnetActivities *testnet.Activity

const TestnetTaskQueue = "TESTNET_TASK_QUEUE"

type TestnetWorkflowOptions struct {
	InstallationID int64
	Owner          string
	Repo           string
	SHA            string
	ChainConfig    types.ChainsConfig
}

func (o *TestnetWorkflowOptions) GenerateCheckOptions(name, status, title, summary, text string, conclusion *string) github.CheckRunOptions {
	return github.CheckRunOptions{
		InstallationID: o.InstallationID,
		Owner:          o.Owner,
		Repo:           o.Repo,
		SHA:            o.SHA,
		Name:           name,
		Status:         stringPtr(status),
		Title:          stringPtr(title),
		Summary:        stringPtr(summary),
		Text:           text,
		Conclusion:     conclusion,
	}
}

func buildTestnetImage(ctx workflow.Context, opts TestnetWorkflowOptions) (string, error) {
	// todo: side effect
	dockerFileBz, err := os.ReadFile(opts.ChainConfig.Image.Dockerfile)

	if err != nil {
		return "", err
	}

	replaces := generateReplace(opts.ChainConfig.Dependencies, opts.Owner, opts.Repo, opts.SHA)

	var builderActivity *builder.Activity
	tag := generateTag(opts.ChainConfig.Name, opts.ChainConfig.Version, opts.Owner, opts.Repo, opts.SHA)

	var builtTag string

	err = workflow.ExecuteActivity(ctx, builderActivity.BuildDockerImage, tag, map[string][]byte{
		"Dockerfile":  dockerFileBz,
		"replaces.sh": replaces,
	}, map[string]string{
		"CHAIN_TAG": opts.ChainConfig.Version,
	}).Get(ctx, &builtTag)

	if err != nil {
		return "", err
	}

	return builtTag, nil
}

func TestnetWorkflow(ctx workflow.Context, opts TestnetWorkflowOptions) (string, error) {
	name := fmt.Sprintf("Testnet (%s) bake", opts.ChainConfig.Name)
	runName := fmt.Sprintf("ib-%s-%s", opts.ChainConfig.Name, opts.SHA[:6])
	start := workflow.Now(ctx)
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}

	ctx = workflow.WithActivityOptions(ctx, options)
	var checkId int64

	output := ""

	err := workflow.ExecuteActivity(ctx, githubActivities.CreateCheck, opts.GenerateCheckOptions(
		name,
		"queued",
		"Launching the testnet",
		"Launching the testnet",
		output,
		nil,
	)).Get(ctx, &checkId)

	if err != nil {
		return "", err
	}

	builtTag, err := buildTestnetImage(ctx, opts)

	if err != nil {
		return "", err
	}

	testnetOptions := testnet.TestnetOptions{
		Name:           runName,
		Image:          builtTag,
		UID:            opts.ChainConfig.Image.UID,
		GID:            opts.ChainConfig.Image.GID,
		BinaryName:     opts.ChainConfig.Image.BinaryName,
		HomeDir:        opts.ChainConfig.Image.HomeDir,
		ValidatorCount: 4,
		NodeCount:      0,
	}

	var providerState string
	err = workflow.ExecuteActivity(ctx, testnetActivities.CreateProvider, testnetOptions).Get(ctx, &providerState)

	if err != nil {
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

	err = workflow.ExecuteActivity(ctx, testnetActivities.LaunchTestnet, testnetOptions).Get(ctx, &chainState)

	if err != nil {
		return "", err
	}

	testnetOptions.ChainState = chainState.ChainState
	testnetOptions.ProviderState = chainState.ProviderState

	markdownTable, err := buildNodeTable(chainState.Nodes)

	if err != nil {
		return "", err
	}

	output += fmt.Sprintf("## Nodes\n%s\n", markdownTable)

	for i := 0; i < 10; i++ {
		var status string
		// TODO: metrics checks
		err = workflow.ExecuteActivity(ctx, testnetActivities.MonitorTestnet, testnetOptions).Get(ctx, &status)

		if err != nil {
			return "", err
		}

		err = workflow.ExecuteActivity(ctx, githubActivities.UpdateCheck, checkId, opts.GenerateCheckOptions(
			name,
			"in_progress",
			fmt.Sprintf("Monitoring the testnet - %d", i),
			fmt.Sprintf("Monitoring the testnet - %d", i),
			output,
			nil,
		)).Get(ctx, nil)

		if err != nil {
			return "", err
		}

		if err := workflow.Sleep(ctx, 10*time.Second); err != nil {
			return "", err
		}
	}

	err = workflow.ExecuteActivity(ctx, githubActivities.UpdateCheck, checkId, opts.GenerateCheckOptions(
		name,
		"in_progress",
		"Shutting down testnet",
		"Shutting down testnet",
		output,
		nil,
	)).Get(ctx, nil)

	if err != nil {
		return "", err
	}

	err = workflow.ExecuteActivity(ctx, githubActivities.UpdateCheck, checkId, opts.GenerateCheckOptions(
		name,
		"completed",
		"The testnet has successfully baked in",
		fmt.Sprintf("The bake in period took %s", workflow.Now(ctx).Sub(start).String()),
		output,
		stringPtr("success"),
	)).Get(ctx, nil)

	if err != nil {
		return "", err
	}

	return "", err
}

func buildNodeTable(nodes []testnet.Node) (string, error) {
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
		return "", nil
	}

	return buf.String(), nil
}
