package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/udhos/boilerplate/awsconfig"
)

type eksClientFactory func(roleArn, region, roleSessionName string) (*eks.Client, error)

func generateEksClient(roleArn, region, roleSessionName string) (*eks.Client, error) {
	options := awsconfig.Options{
		Region:          region,
		RoleArn:         roleArn,
		RoleSessionName: roleSessionName,
	}
	awsCfg, errCfg := awsconfig.AwsConfig(options)
	if errCfg != nil {
		return nil, fmt.Errorf("generateEksClient: could not get aws config: %w", errCfg)
	}

	eksClient := eks.NewFromConfig(awsCfg.AwsConfig)

	return eksClient, nil
}
