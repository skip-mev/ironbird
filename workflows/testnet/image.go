package testnet

import (
	"fmt"
	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/messages"
	"go.temporal.io/sdk/workflow"
	"os"
)

func generateReplace(dependencies map[string]string, owner, repo, tag string) []byte {
	orig := dependencies[fmt.Sprintf("%s/%s", owner, repo)]

	return []byte(fmt.Sprintf("go mod edit -replace %s=github.com/%s/%s@%s", orig, owner, repo, tag))
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

	replaces := generateReplace(req.ChainConfig.Dependencies, req.Owner, req.Repo, req.SHA)

	var builderActivity *builder.Activity
	tag := generateTag(req.ChainConfig.Name, req.ChainConfig.Version, req.Owner, req.Repo, req.SHA)

	var buildResult messages.BuildDockerImageResponse

	err = workflow.ExecuteActivity(ctx, builderActivity.BuildDockerImage, messages.BuildDockerImageRequest{
		Tag: tag,
		Files: map[string][]byte{
			"Dockerfile":  dockerFileBz,
			"replaces.sh": replaces,
		},
		BuildArguments: map[string]string{
			"CHAIN_TAG": req.ChainConfig.Version,
		},
	}).Get(ctx, &buildResult)

	if err != nil {
		return messages.BuildDockerImageResponse{}, err
	}

	return buildResult, nil
}
