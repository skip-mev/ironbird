package util

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

func FetchDockerRepoToken(ctx context.Context, awsCfg aws.Config) (string, error) {
	ecrClient := ecr.NewFromConfig(awsCfg, func(options *ecr.Options) {
		options.Region = "us-east-1"
	})

	token, err := ecrClient.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	} else if len(token.AuthorizationData) == 0 {
		return "", fmt.Errorf("no authorization token found")
	}

	return *token.AuthorizationData[0].AuthorizationToken, nil
}
