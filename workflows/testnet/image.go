package testnet

import (
	"fmt"
	"os"

	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/messages"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	SDK_REPO          = "cosmos-sdk"
	IRONBIRD_SDK_REPO = "ironbird-cosmos-sdk"
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

	var buildResult messages.BuildDockerImageResponse

	buildArguments := make(map[string]string)
	buildArguments["GIT_SHA"] = generateTag(req.ChainConfig.Name, req.ChainConfig.Version, req.Owner, req.Repo, req.SHA)

	if req.Repo == SDK_REPO || req.Repo == IRONBIRD_SDK_REPO {
		buildArguments["CHAIN_TAG"] = req.SHA
		buildArguments["CHAIN_SRC"] = fmt.Sprintf("https://github.com/%s/%s", req.Owner, req.Repo)
	} else {
		buildArguments["CHAIN_TAG"] = req.ChainConfig.Version
		buildArguments["REPLACE_CMD"] = generateReplace(req.ChainConfig.Dependencies, req.Owner, req.Repo, req.SHA)
		logger.Info("replaces", zap.String("", buildArguments["REPLACE_CMD"]))
	}

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
