package builder

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
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

	"github.com/skip-mev/ironbird/util"
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
	dependencies = map[string]string{
		"cometbft/cometbft": "github.com/cometbft/cometbft",
		"cosmos/cosmos-sdk": "github.com/cosmos/cosmos-sdk",
		"cosmos/evm":        "github.com/cosmos/evm",
	}
	repoOwners = map[string]string{
		"cometbft":   "cometbft",
		"cosmos-sdk": "cosmos",
		"gaia":       "cosmos",
		"evm":        "cosmos",
	}
)

func (a *Activity) getAuthenticationToken(ctx context.Context) (string, string, error) {
	token, err := util.FetchDockerRepoToken(ctx, *a.AwsConfig)
	if err != nil {
		return "", "", err
	}

	decodedToken, err := base64.StdEncoding.DecodeString(token)

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

	ecrClient := ecr.NewFromConfig(*a.AwsConfig, func(options *ecr.Options) {
		options.Region = "us-east-2"
	})

	repositories, err := ecrClient.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
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

	_, err = ecrClient.CreateRepository(ctx, &ecr.CreateRepositoryInput{
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

func generateTag(imageName, version, repo, sha string) string {
	if repo == "cometbft" {
		return fmt.Sprintf("%s-%s-%s-%s", imageName, version, repo, sha)
	}
	return fmt.Sprintf("%s-%s", repo, sha)
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

	image, exists := a.Chains[req.ImageConfig.Image]
	if !exists {
		return messages.BuildDockerImageResponse{}, fmt.Errorf("image config not found for %s", req.ImageConfig.Image)
	}

	dockerfileContent, err := os.ReadFile(image.Dockerfile)
	if err != nil {
		return messages.BuildDockerImageResponse{}, fmt.Errorf("failed to read dockerfile from %s: %w", image.Dockerfile, err)
	}

	fs := staticfs.NewFS()

	fs.Add("Dockerfile", &fstypes.Stat{Mode: 0644}, dockerfileContent)

	for _, additionalFile := range image.AdditionalFiles {
		baseName := filepath.Base(additionalFile)
		fileContent, err := os.ReadFile(additionalFile)
		if err != nil {
			return messages.BuildDockerImageResponse{}, fmt.Errorf("failed to read file %s: %w", additionalFile, err)
		}
		fs.Add(baseName, &fstypes.Stat{Mode: 0644}, fileContent)
	}

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
		"platform": "linux/amd64",
	}

	buildArguments := make(map[string]string)

	tag := generateTag(req.ImageConfig.Image, req.ImageConfig.Version, req.Repo, req.SHA)
	buildArguments["GIT_SHA"] = tag

	// When load testing CometBFT, we build a simapp image using a specified SDK version, and then edit go.mod to replace
	// CometBFT with the specified commit SHA
	if req.Repo == "cometbft" {
		buildArguments["CHAIN_SRC"] = "https://github.com/cosmos/cosmos-sdk"
		buildArguments["CHAIN_TAG"] = req.ImageConfig.Version
		buildArguments["REPLACE_CMD"] = generateReplace(dependencies, repoOwners[req.Repo], req.Repo, req.SHA)
	} else {
		buildArguments["CHAIN_TAG"] = req.SHA
		buildArguments["CHAIN_SRC"] = fmt.Sprintf("https://github.com/%s/%s", repoOwners[req.Repo], req.Repo)
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
