package testnet

import (
	"fmt"
	"os"
	"slices"

	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/messages"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

var (
	// SKIP_REPLACE_REPOS are repositories where ironbird does not need to run the replace workflow
	// as checking out to the chain branch tag is sufficient to test the intended changes
	// (e.g. cosmos-sdk repo does not need to replace a dependency, just to run simapp using the SDK version
	// based on the commit SHA passed to ironbird)
	SKIP_REPLACE_REPOS = []string{"cosmos-sdk", "ironbird-cosmos-sdk", "gaia"}
)

func generateReplace(dependencies map[string]string, owner, repo, tag string) string {
	orig := dependencies[fmt.Sprintf("%s/%s", owner, repo)]

	return fmt.Sprintf("go mod edit -replace github.com/%s=github.com/%s/%s@%s", orig, owner, repo, tag)
}

func generateTag(chain, version, owner, repo, sha string) string {
	return fmt.Sprintf("%s%s-%s%s-%s", chain, version, owner, repo, sha)
}

func buildImage(ctx workflow.Context, req messages.TestnetWorkflowRequest) (messages.BuildDockerImageResponse, error) {
	logger, _ := zap.NewDevelopment()

	// todo: side effect
	dockerFileBz, err := os.ReadFile(req.ChainConfig.Image.Dockerfile)

	if err != nil {
		return messages.BuildDockerImageResponse{}, err
	}

	var builderActivity *builder.Activity
	tag := generateTag(req.ChainConfig.Name, req.ChainConfig.Version, req.Owner, req.Repo, req.SHA)

	var buildResult messages.BuildDockerImageResponse

	var chainTag string
	replaces := ""
	// Skip replace script in the SDK repo because its not needed
	if slices.Contains(SKIP_REPLACE_REPOS, req.Repo) {
		chainTag = req.SHA
	} else {
		chainTag = req.ChainConfig.Version
		replaces = generateReplace(req.ChainConfig.Dependencies, req.Owner, req.Repo, req.SHA)
		logger.Info("replaces", zap.String("", replaces))
	}

	err = workflow.ExecuteActivity(ctx, builderActivity.BuildDockerImage, messages.BuildDockerImageRequest{
		Tag: tag,
		Files: map[string][]byte{
			"Dockerfile": dockerFileBz,
		},
		BuildArguments: map[string]string{
			"CHAIN_TAG":   chainTag,
			"GIT_SHA":     tag,
			"REPLACE_CMD": replaces,
		},
	}).Get(ctx, &buildResult)

	if err != nil {
		return messages.BuildDockerImageResponse{}, err
	}

	return buildResult, nil
}
