package util

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
)

func FetchDockerRepoToken(ctx context.Context, awsCfg aws.Config) (string, error) {
	ecrClient := ecrpublic.NewFromConfig(awsCfg, func(options *ecrpublic.Options) {
		// ecrpublic only works in us-east-1
		options.Region = "us-east-1"
	})

	token, err := ecrClient.GetAuthorizationToken(ctx, &ecrpublic.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	} else if token.AuthorizationData == nil {
		return "", fmt.Errorf("no authorization token found")
	}

	return *token.AuthorizationData.AuthorizationToken, nil
}
