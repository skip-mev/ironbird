package builder

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecrpublic/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/staticfs"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"github.com/tonistiigi/fsutil"
	fstypes "github.com/tonistiigi/fsutil/types"
)

type Activity struct {
	BuilderConfig types.BuilderConfig
	AwsConfig     *aws.Config
	Chains        types.Chains
}

type BuildResult struct {
	FQDNTag string
	Logs    []byte
}

var (
	// SKIP_REPLACE_REPOS are repositories where ironbird does not need to run the replace workflow
	// as checking out to the chain branch tag is sufficient to test the intended changes
	// (e.g. cosmos-sdk repo does not need to replace a dependency, just to run simapp using the SDK version
	// based on the commit SHA passed to ironbird. To test cometbft on the other hand, we use a base simapp image
	// and then replace the cometbft dependency with the intended commit version)
	SKIP_REPLACE_REPOS = []string{"cosmos-sdk", "ironbird-cosmos-sdk", "gaia"}
	dependencies       = map[string]string{
		"skip-mev/ironbird-cometbft":   "github.com/cometbft/cometbft",
		"skip-mev/ironbird-cosmos-sdk": "github.com/cosmos/cosmos-sdk",
		"cometbft/cometbft":            "github.com/cometbft/cometbft",
		"cosmos/cosmos-sdk":            "github.com/cosmos/cosmos-sdk",
	}
	repoOwners = map[string]string{
		"ironbird-cometbft":   "skip-mev",
		"ironbird-cosmos-sdk": "skip-mev",
		"cometbft":            "cometbft",
		"cosmos-sdk":          "cosmos",
		"gaia":                "cosmos",
	}
)

func (a *Activity) getAuthenticationToken(ctx context.Context) (string, string, error) {
	ecrClient := ecrpublic.NewFromConfig(*a.AwsConfig, func(options *ecrpublic.Options) {
		// ecrpublic only works in us-east-1
		options.Region = "us-east-1"
	})

	token, err := ecrClient.GetAuthorizationToken(ctx, &ecrpublic.GetAuthorizationTokenInput{})

	if err != nil {
		return "", "", err
	}

	if token.AuthorizationData.AuthorizationToken == nil {
		return "", "", fmt.Errorf("no authorization token found")
	}

	decodedToken, err := base64.StdEncoding.DecodeString(*token.AuthorizationData.AuthorizationToken)

	if err != nil {
		return "", "", fmt.Errorf("failed to decode token: %w", err)
	}

	// username:string
	parts := strings.Split(string(decodedToken), ":")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token format")
	}

	return parts[0], parts[1], nil
}

func (a *Activity) createRepositoryIfNotExists(ctx context.Context) error {
	stsClient := sts.NewFromConfig(*a.AwsConfig)
	stsIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})

	if err != nil {
		return fmt.Errorf("failed to fetch STS identity: %w", err)
	}

	ecrClient := ecrpublic.NewFromConfig(*a.AwsConfig, func(options *ecrpublic.Options) {
		// ecrpublic only works in us-east-1
		options.Region = "us-east-1"
	})

	repositories, err := ecrClient.DescribeRepositories(ctx, &ecrpublic.DescribeRepositoriesInput{
		RepositoryNames: []string{
			a.BuilderConfig.Registry.ImageName,
		},
		RegistryId: stsIdentity.Account,
	})

	var notFoundErr *ecrtypes.RepositoryNotFoundException

	if err != nil && !errors.As(err, &notFoundErr) {
		return err
	}

	if repositories != nil && len(repositories.Repositories) != 0 {
		return nil
	}

	_, err = ecrClient.CreateRepository(ctx, &ecrpublic.CreateRepositoryInput{
		RepositoryName: aws.String(a.BuilderConfig.Registry.ImageName),
	})

	if err != nil {
		return err
	}

	return nil
}

func generateReplace(dependencies map[string]string, owner, repo, tag string) string {
	orig := dependencies[fmt.Sprintf("%s/%s", owner, repo)]
	return fmt.Sprintf("go mod edit -replace github.com/%s=github.com/%s/%s@%s", orig, owner, repo, tag)
}

func generateTag(chain, version, owner, repo, sha string) string {
	return fmt.Sprintf("%s%s-%s%s-%s", chain, version, owner, repo, sha)
}

func (a *Activity) BuildDockerImage(ctx context.Context, req messages.BuildDockerImageRequest) (messages.BuildDockerImageResponse, error) {
	logger, _ := zap.NewDevelopment()
	if err := a.createRepositoryIfNotExists(ctx); err != nil {
		return messages.BuildDockerImageResponse{}, err
	}

	username, password, err := a.getAuthenticationToken(ctx)
	if err != nil {
		return messages.BuildDockerImageResponse{}, err
	}

	bkClient, err := client.New(ctx, a.BuilderConfig.BuildKitAddress)
	if err != nil {
		return messages.BuildDockerImageResponse{}, err
	}
	defer bkClient.Close()

	image, exists := a.Chains[req.ChainConfig.Image]
	if !exists {
		return messages.BuildDockerImageResponse{}, fmt.Errorf("image config not found for %s", req.ChainConfig.Image)
	}

	dockerfileContent, err := os.ReadFile(image.Dockerfile)
	if err != nil {
		return messages.BuildDockerImageResponse{}, fmt.Errorf("failed to read dockerfile from %s: %w", image.Dockerfile, err)
	}

	fs := staticfs.NewFS()

	fs.Add("Dockerfile", &fstypes.Stat{Mode: 0644}, dockerfileContent)

	authProvider := authprovider.NewDockerAuthProvider(&configfile.ConfigFile{
		AuthConfigs: map[string]configtypes.AuthConfig{
			a.BuilderConfig.Registry.URL: {
				Username: username,
				Password: password,
			},
		},
	}, map[string]*authprovider.AuthTLSConfig{})

	frontendAttrs := map[string]string{
		"filename": "Dockerfile",
		"target":   "",
	}

	buildArguments := make(map[string]string)
	buildArguments["GIT_SHA"] = generateTag(req.ChainConfig.Name, image.Version, repoOwners[req.Repo], req.Repo, req.SHA)
	tag := generateTag(req.ChainConfig.Name, image.Version, "", req.Repo, req.SHA)

	if slices.Contains(SKIP_REPLACE_REPOS, req.Repo) {
		buildArguments["CHAIN_TAG"] = req.SHA
		buildArguments["CHAIN_SRC"] = fmt.Sprintf("https://github.com/%s/%s", repoOwners[req.Repo], req.Repo)
	} else {
		buildArguments["CHAIN_TAG"] = image.Version
		buildArguments["REPLACE_CMD"] = generateReplace(dependencies, repoOwners[req.Repo], req.Repo, req.SHA)
	}

	for k, v := range buildArguments {
		frontendAttrs[fmt.Sprintf("build-arg:%s", k)] = v
	}

	logger.Info("building docker image", zap.Any("build_arguments", buildArguments),
		zap.Any("frontend_attrs", frontendAttrs), zap.String("dockerfile_path", image.Dockerfile))

	fqdnTag := fmt.Sprintf("%s/%s:%s", a.BuilderConfig.Registry.URL, a.BuilderConfig.Registry.ImageName, tag)
	solveOpt := client.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,
		LocalMounts: map[string]fsutil.FS{
			"context":    fs,
			"dockerfile": fs,
		},
		Session: []session.Attachable{
			authProvider,
		},
		Exports: []client.ExportEntry{
			{
				Type: client.ExporterImage,
				Attrs: map[string]string{
					"name": fqdnTag,
					"push": "true",
				},
			},
		},
	}

	statusChan := make(chan *client.SolveStatus)
	var logs bytes.Buffer

	go func() {
		for status := range statusChan {
			for _, v := range status.Logs {
				logLine := fmt.Sprintf("[%s]: %s\n", v.Timestamp.String(), string(v.Data))
				logs.WriteString(logLine)
				fmt.Print(logLine)
			}
		}
	}()

	_, err = bkClient.Solve(ctx, nil, solveOpt, statusChan)
	if err != nil {
		return messages.BuildDockerImageResponse{}, err
	}

	return messages.BuildDockerImageResponse{
		FQDNTag: fqdnTag,
		Logs:    logs.Bytes(),
	}, nil
}
