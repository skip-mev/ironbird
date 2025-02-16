package testnet

import (
	"fmt"
	"github.com/skip-mev/ironbird/activities/builder"
	"go.temporal.io/sdk/workflow"
	"os"
)

func generateReplace(dependencies map[string]string, owner, repo, tag string) []byte {
	orig := dependencies[fmt.Sprintf("%s/%s", owner, repo)]

	return []byte(fmt.Sprintf("go mod edit -replace %s=github.com/%s/%s@%s", orig, owner, repo, tag))
}

func generateTag(chain, version, owner, repo, sha string) string {
	return fmt.Sprintf("ironbird:%s%s-%s%s-%s", chain, version, owner, repo, sha)
}

func buildImage(ctx workflow.Context, opts WorkflowOptions) (string, error) {
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
