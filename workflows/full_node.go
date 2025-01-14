package workflows

import (
	"fmt"
	"github.com/skip-mev/ironbird/activities/fullnode"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/builder"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/petri/core/v2/provider/digitalocean"
	"go.temporal.io/sdk/workflow"
	"os"
	"time"
)

// temporal dependency injection
var githubActivities *github.NotifierActivity
var nodeActivities *fullnode.NodeActivity

const FullNodeTaskQueue = "FULL_NODE_TASK_QUEUE"

type FullNodeWorkflowOptions struct {
	InstallationID int64
	Owner          string
	Repo           string
	SHA            string
	ChainConfig    types.ChainsConfig
}

func (o *FullNodeWorkflowOptions) GenerateCheckOptions(name, status, title, summary string, conclusion *string) github.CheckRunOptions {
	return github.CheckRunOptions{
		InstallationID: o.InstallationID,
		Owner:          o.Owner,
		Repo:           o.Repo,
		SHA:            o.SHA,
		Name:           name,
		Status:         stringPtr(status),
		Title:          stringPtr(title),
		Summary:        stringPtr(summary),
		Conclusion:     conclusion,
	}
}

func buildImage(ctx workflow.Context, opts FullNodeWorkflowOptions) (string, error) {
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

func FullNodeWorkflow(ctx workflow.Context, opts FullNodeWorkflowOptions) (string, error) {
	name := fmt.Sprintf("Full node (%s) mainnet bake", opts.ChainConfig.Name)
	start := workflow.Now(ctx)
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}

	ctx = workflow.WithActivityOptions(ctx, options)
	var checkId int64

	err := workflow.ExecuteActivity(ctx, githubActivities.CreateCheck, opts.GenerateCheckOptions(
		name,
		"queued",
		"Launching the full node",
		"Launching the full node",
		nil,
	)).Get(ctx, &checkId)

	if err != nil {
		return "", err
	}

	defer workflow.ExecuteActivity(ctx, nodeActivities.ShutdownNode, workflow.GetInfo(ctx).OriginalRunID)

	var result string

	// TODO(Zygimantass): get these out of here
	taskConfig := digitalocean.DigitalOceanTaskConfig{
		"region":   "ams3",
		"size":     "s-2vcpu-4gb",
		"image_id": "170910128",
	}

	builtTag, err := buildImage(ctx, opts)

	if err != nil {
		return "", err
	}

	err = workflow.ExecuteActivity(ctx, nodeActivities.LaunchNode, fullnode.NodeOptions{
		Name:                    workflow.GetInfo(ctx).OriginalRunID,
		Image:                   builtTag,
		SnapshotURL:             opts.ChainConfig.SnapshotURL,
		UID:                     opts.ChainConfig.Image.UID,
		GID:                     opts.ChainConfig.Image.GID,
		BinaryName:              opts.ChainConfig.Image.BinaryName,
		HomeDir:                 opts.ChainConfig.Image.HomeDir,
		GasPrices:               opts.ChainConfig.Image.GasPrices,
		ProviderSpecificOptions: taskConfig,
	}).Get(ctx, &result)

	if err != nil {
		return "", err
	}

	for i := 0; i < 10; i++ {
		var status string
		// TODO: metrics checks
		err = workflow.ExecuteActivity(ctx, nodeActivities.MonitorContainer, workflow.GetInfo(ctx).OriginalRunID, result).Get(ctx, &status)

		if err != nil {
			return "", err
		}

		err = workflow.ExecuteActivity(ctx, githubActivities.UpdateCheck, checkId, opts.GenerateCheckOptions(
			name,
			"in_progress",
			fmt.Sprintf("Monitoring the node - %d", i),
			fmt.Sprintf("Monitoring the node - %d", i),
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
		"Shutting down node",
		"Shutting down node",
		nil,
	)).Get(ctx, nil)

	if err != nil {
		return "", err
	}

	err = workflow.ExecuteActivity(ctx, githubActivities.UpdateCheck, checkId, opts.GenerateCheckOptions(
		name,
		"completed",
		"The full node has successfully baked in",
		fmt.Sprintf("The bake in period took %s", workflow.Now(ctx).Sub(start).String()),
		stringPtr("success"),
	)).Get(ctx, nil)

	if err != nil {
		return "", err
	}

	return result, err
}

func generateReplace(dependencies map[string]string, owner, repo, tag string) []byte {
	orig := dependencies[fmt.Sprintf("%s/%s", owner, repo)]

	return []byte(fmt.Sprintf("go mod edit -replace %s=github.com/%s/%s@%s", orig, owner, repo, tag))
}

func generateTag(chain, version, owner, repo, sha string) string {
	return fmt.Sprintf("ironbird:%s-%s-%s-%s-%s", chain, version, owner, repo, sha)
}
