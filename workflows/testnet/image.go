package testnet

import (
	"fmt"
	"os"

	"github.com/skip-mev/ironbird/activities/builder"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	SDK_REPO = "cosmos-sdk"
)

func generateReplace(dependencies map[string]string, owner, repo, tag string) string {
	orig := dependencies[fmt.Sprintf("%s/%s", owner, repo)]

	return fmt.Sprintf("go mod edit -replace github.com/%s=github.com/%s/%s@%s", orig, owner, repo, tag)
}

func generateTag(chain, version, owner, repo, sha string) string {
	return fmt.Sprintf("%s%s-%s%s-%s", chain, version, owner, repo, sha)
}

func buildImage(ctx workflow.Context, opts WorkflowOptions) (builder.BuildResult, error) {
	logger, _ := zap.NewDevelopment()

	// todo: side effect
	dockerFileBz, err := os.ReadFile(opts.ChainConfig.Image.Dockerfile)

	if err != nil {
		return builder.BuildResult{}, err
	}

	logger.Info("opts", zap.Any("opts", opts))
	logger.Info("dockerfile", zap.String("", string(dockerFileBz)))

	var builderActivity *builder.Activity
	tag := generateTag(opts.ChainConfig.Name, opts.ChainConfig.Version, opts.Owner, opts.Repo, opts.SHA)

	var buildResult builder.BuildResult

	var chainTag string
	replaces := ""
	// Skip replace script in the SDK repo because its not needed
	if opts.Repo == SDK_REPO {
		chainTag = opts.SHA
	} else {
		chainTag = opts.ChainConfig.Version
		replaces = generateReplace(opts.ChainConfig.Dependencies, opts.Owner, opts.Repo, opts.SHA)
		logger.Info("replaces", zap.String("", replaces))
	}

	err = workflow.ExecuteActivity(ctx, builderActivity.BuildDockerImage, tag, map[string][]byte{
		"Dockerfile": dockerFileBz,
	}, map[string]string{
		"CHAIN_TAG":   chainTag,
		"GIT_SHA":     tag,
		"REPLACE_CMD": replaces,
	}).Get(ctx, &buildResult)

	if err != nil {
		return builder.BuildResult{}, err
	}

	return buildResult, nil
}
