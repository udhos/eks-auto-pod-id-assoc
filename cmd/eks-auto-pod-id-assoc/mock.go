package main

import (
	"errors"

	"github.com/segmentio/ksuid"
)

// This file should be moved to app_test.go, but for now we keep it here
// in order to run it from main.go

func newMockClient() *mockClient {
	client := &mockClient{
		regions: map[string][]mockCluster{
			"us-east-1": {
				{
					clusterName: "example-cluster-2",
					serviceAccounts: []serviceAccount{
						{Name: "sa1", Namespace: "default", AwsRoleArn: "arn:aws:iam::123456789012:role/sa1-role"},
					},
					podIdentityAssociations: []podIdentityAssociation{
						{
							AssociationID:      "example-assoc-id-1",
							ClusterName:        "example-cluster-2",
							ServiceAccountName: "sa1",
							RoleArn:            "arn:aws:iam::123456789012:role/sa1-role",
						},
					},
				},
			},
			"sa-east-1": {
				{
					clusterName: "my-eks-cluster",
					serviceAccounts: []serviceAccount{
						{Name: "sa1", Namespace: "default", AwsRoleArn: "arn:aws:iam::123456789012:role/sa1-role"},
						{Name: "sa2", Namespace: "kube-system", AwsRoleArn: "arn:aws:iam::123456789012:role/sa2-role"},
					},
					podIdentityAssociations: []podIdentityAssociation{
						{
							AssociationID:      "example-assoc-id-1",
							ClusterName:        "my-eks-cluster",
							ServiceAccountName: "sa1",
							RoleArn:            "arn:aws:iam::123456789012:role/sa1-role",
						},
						{
							AssociationID:      "example-assoc-id-2",
							ClusterName:        "my-eks-cluster",
							ServiceAccountName: "sa2",
							RoleArn:            "arn:aws:iam::123456789012:role/sa2-role",
						},
					},
				},
				{
					clusterName: "another-cluster",
				},
			},
		},
	}
	return client
}

type mockClient struct {
	regions map[string][]mockCluster // region -> clusters
}

type mockCluster struct {
	clusterName             string
	serviceAccounts         []serviceAccount
	podIdentityAssociations []podIdentityAssociation
}

func (c *mockClient) listEKSClusters(roleArn, region string) ([]string, error) {
	clusters := c.regions[region]
	var clusterNames []string
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.clusterName)
	}
	return clusterNames, nil
}

func (c *mockClient) listServiceAccounts(roleArn, region,
	clusterName string) ([]serviceAccount, error) {
	for _, cluster := range c.regions[region] {
		if cluster.clusterName == clusterName {
			return cluster.serviceAccounts, nil
		}
	}
	return nil, errors.New("cluster not found")
}

func (c *mockClient) listPodIdentityAssociations(roleArn, region,
	clusterName string) ([]podIdentityAssociation, error) {
	for _, cluster := range c.regions[region] {
		if cluster.clusterName == clusterName {
			return cluster.podIdentityAssociations, nil
		}
	}
	return nil, errors.New("cluster not found")
}

func (c *mockClient) createPodIdentityAssociation(roleArn, region,
	clusterName, serviceAccountName, serviceAccountRoleArn string) error {

	cluster, err := c.findCluster(region, clusterName)
	if err != nil {
		return err
	}

	_, errAssoc := c.findPodIdentityAssociationByServiceAccount(cluster, serviceAccountName)
	if errAssoc == nil {
		return errors.New("pod identity association already exists for service account")
	}

	id := ksuid.New().String()

	associationID := "assoc-" + id

	newAssoc := podIdentityAssociation{
		AssociationID:      associationID,
		ClusterName:        clusterName,
		ServiceAccountName: serviceAccountName,
		RoleArn:            serviceAccountRoleArn,
	}

	cluster.podIdentityAssociations = append(cluster.podIdentityAssociations, newAssoc)

	return nil
}

func (c *mockClient) findCluster(region, clusterName string) (*mockCluster, error) {
	clusters := c.regions[region]
	for i, cluster := range clusters {
		if cluster.clusterName == clusterName {
			return &c.regions[region][i], nil
		}
	}
	return nil, errors.New("cluster not found")
}

func (c *mockClient) findPodIdentityAssociationByServiceAccount(cluster *mockCluster,
	serviceAccountName string) (podIdentityAssociation, error) {
	for _, assoc := range cluster.podIdentityAssociations {
		if assoc.ServiceAccountName == serviceAccountName {
			return assoc, nil
		}
	}
	return podIdentityAssociation{}, errors.New("pod identity association not found")
}

func (c *mockClient) deletePodIdentityAssociation(roleArn, region,
	clusterName, associationID string) error {

	cluster, err := c.findCluster(region, clusterName)
	if err != nil {
		return err
	}

	for i, assoc := range cluster.podIdentityAssociations {
		if assoc.AssociationID == associationID {
			cluster.podIdentityAssociations = append(cluster.podIdentityAssociations[:i],
				cluster.podIdentityAssociations[i+1:]...)
			return nil
		}
	}

	return errors.New("pod identity association not found")
}
