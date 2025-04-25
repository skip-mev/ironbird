package builder

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

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
	"github.com/skip-mev/ironbird/types"
	"github.com/tonistiigi/fsutil"
	fstypes "github.com/tonistiigi/fsutil/types"
	"go.uber.org/zap"
)

type Activity struct {
	BuilderConfig types.BuilderConfig
	AwsConfig     *aws.Config
}

type BuildResult struct {
	FQDNTag string
	Logs    []byte
}

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

func (a *Activity) BuildDockerImage(ctx context.Context, tag string, files map[string][]byte, buildArgs map[string]string) (BuildResult, error) {
	logger, _ := zap.NewDevelopment()
	if err := a.createRepositoryIfNotExists(ctx); err != nil {
		return BuildResult{}, err
	}

	username, password, err := a.getAuthenticationToken(ctx)

	if err != nil {
		return BuildResult{}, err
	}

	bkClient, err := client.New(ctx, a.BuilderConfig.BuildKitAddress)

	if err != nil {
		return BuildResult{}, err
	}
	defer bkClient.Close()

	fs := staticfs.NewFS()
	for name, content := range files {
		fs.Add(name, &fstypes.Stat{Mode: 0644}, content)
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
	}

	for k, v := range buildArgs {
		frontendAttrs[fmt.Sprintf("build-arg:%s", k)] = v
	}

	fqdnTag := fmt.Sprintf("%s/%s:%s", a.BuilderConfig.Registry.URL, a.BuilderConfig.Registry.ImageName, tag)

	logger.Info("fqdntag", zap.String("fqdnTag", fqdnTag))
	logger.Info("BuilderConfig.Registry.URL", zap.String("", a.BuilderConfig.Registry.URL))
	logger.Info("a.BuilderConfig.Registry.ImageName", zap.String("", a.BuilderConfig.Registry.ImageName))
	logger.Info("tag", zap.String("tag", tag))

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
		return BuildResult{}, err
	}

	return BuildResult{
		FQDNTag: fqdnTag,
		Logs:    logs.Bytes(),
	}, nil
}
