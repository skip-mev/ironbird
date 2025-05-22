package testnet

import (
	"fmt"
	"os"
	"slices"

	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/messages"
	"go.temporal.io/sdk/workflow"
)

var (
	// SKIP_REPLACE_REPOS are repositories where ironbird does not need to run the replace workflow
	// as checking out to the chain branch tag is sufficient to test the intended changes
	// (e.g. cosmos-sdk repo does not need to replace a dependency, just to run simapp using the SDK version
	// based on the commit SHA passed to ironbird. To test cometbft on the other hand, we use a base simapp image
	// and then replace the cometbft dependency with the intended commit version)
	SKIP_REPLACE_REPOS = []string{"cosmos-sdk", "ironbird-cosmos-sdk", "gaia"}
	dependencies = map[string]string{
		"ironbird-cometbft":   "github.com/cometbft/cometbft",
		"ironbird-cosmos-sdk": "github.com/cosmos/cosmos-sdk",
		"cometbft":            "github.com/cometbft/cometbft",
		"cosmos-sdk":          "github.com/cosmos/cosmos-sdk",
	}
	repoOwners = map[string]string{
		"ironbird-cometbft":   "skip-mev",
		"ironbird-cosmos-sdk": "skip-mev",
		"cometbft":            "cometbft",
		"cosmos-sdk":          "cosmos",
	}
)

func generateReplace(dependencies map[string]string, owner, repo, tag string) string {
	orig := dependencies[fmt.Sprintf("%s/%s", owner, repo)]

	return fmt.Sprintf("go mod edit -replace github.com/%s=github.com/%s/%s@%s", orig, owner, repo, tag)
}

func generateTag(chain, version, owner, repo, sha string) string {
	return fmt.Sprintf("%s%s-%s%s-%s", chain, version, owner, repo, sha)
}

func buildImage(ctx workflow.Context, req messages.TestnetWorkflowRequest) (messages.BuildDockerImageResponse, error) {
	// todo: side effect
	dockerFileBz, err := os.ReadFile(req.ChainConfig.Image.Dockerfile)

	if err != nil {
		return messages.BuildDockerImageResponse{}, err
	}

	var builderActivity *builder.Activity

	var buildResult messages.BuildDockerImageResponse
	buildArguments := make(map[string]string)
	buildArguments["GIT_SHA"] = generateTag(req.ChainConfig.Name, req.ChainConfig.Version, repoOwners[req.Repo], req.Repo, req.SHA)

	if slices.Contains(SKIP_REPLACE_REPOS, req.Repo) {
		buildArguments["CHAIN_TAG"] = req.SHA
		buildArguments["CHAIN_SRC"] = fmt.Sprintf("https://github.com/%s/%s", repoOwners[req.Repo], req.Repo)
	} else {
		buildArguments["CHAIN_TAG"] = req.ChainConfig.Version
		buildArguments["REPLACE_CMD"] = generateReplace(dependencies, repoOwners[req.Repo], req.Repo, req.SHA)
	}

	logger := workflow.GetLogger(ctx)
	logger.Info("building docker image", zap.Any("build_arguments", buildArguments))

	err = workflow.ExecuteActivity(ctx, builderActivity.BuildDockerImage, messages.BuildDockerImageRequest{
		Tag: buildArguments["GIT_SHA"],
		Files: map[string][]byte{
			"Dockerfile": dockerFileBz,
		},
		BuildArguments: buildArguments,
	}).Get(ctx, &buildResult)

	if err != nil {
		return messages.BuildDockerImageResponse{}, err
	}

	return buildResult, nil
}
