package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/udhos/boilerplate/awsconfig"
)

func newRealClient(prog string, dry bool) *realClient {
	return &realClient{
		prog: prog,
		dry:  dry}
}

type realClient struct {
	prog string
	dry  bool
}

func (c *realClient) listEKSClusters(roleArn,
	region string) ([]string, error) {

	const me = "listEKSClusters"

	options := awsconfig.Options{
		Region:          region,
		RoleArn:         roleArn,
		RoleSessionName: c.prog,
	}
	awsCfg, errCfg := awsconfig.AwsConfig(options)
	if errCfg != nil {
		return nil, fmt.Errorf("%s: could not get aws config: %w", me, errCfg)
	}

	clientEks := eks.NewFromConfig(awsCfg.AwsConfig)

	var maxResults int32 = 100 // 1..100

	paginator := eks.NewListClustersPaginator(clientEks, &eks.ListClustersInput{
		MaxResults: aws.Int32(maxResults),
	})

	var clusterNames []string

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%s: failed to get page: %w", me, err)
		}
		clusterNames = append(clusterNames, page.Clusters...)
	}

	infof("%s: region=%q found clusters: %d", me, region, len(clusterNames))

	return clusterNames, nil
}

func (c *realClient) listServiceAccounts(self bool, roleArn, region,
	clusterName string) ([]serviceAccount, error) {
	return []serviceAccount{}, errors.New("not implemented")
}

func (c *realClient) listPodIdentityAssociations(self bool, roleArn, region,
	clusterName string) ([]podIdentityAssociation, error) {
	return []podIdentityAssociation{}, errors.New("not implemented")
}

func (c *realClient) createPodIdentityAssociation(self bool, roleArn, region,
	clusterName, serviceAccountName, serviceAccountRoleArn string) error {
	return errors.New("not implemented")
}

func (c *realClient) deletePodIdentityAssociation(self bool, roleArn, region,
	clusterName, associationID string) error {
	return errors.New("not implemented")
}
