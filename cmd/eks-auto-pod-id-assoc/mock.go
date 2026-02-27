package main

import "errors"

// This file should be moved to app_test.go, but for now we keep it here
// in order to run it from main.go

func newMockClient() *mockClient {
	client := &mockClient{
		clusters: map[string][]mockCluster{
			"us-east-1": {
				{
					clusterName: "example-cluster-2",
					serviceAccounts: map[string][]serviceAccount{
						"example-cluster-2": {
							{Name: "sa1", Namespace: "default", AwsRoleArn: "arn:aws:iam::123456789012:role/sa1-role"},
						},
					},
					podIdentityAssociations: map[string][]podIdentityAssociation{
						"example-cluster-2": {
							{ClusterName: "example-cluster-2", ServiceAccountName: "sa1", RoleArn: "arn:aws:iam::123456789012:role/sa1-role"},
						},
					},
				},
			},
			"sa-east-1": {
				{
					clusterName: "my-eks-cluster",
					serviceAccounts: map[string][]serviceAccount{
						"my-eks-cluster": {
							{Name: "sa1", Namespace: "default", AwsRoleArn: "arn:aws:iam::123456789012:role/sa1-role"},
							{Name: "sa2", Namespace: "kube-system", AwsRoleArn: "arn:aws:iam::123456789012:role/sa2-role"},
						},
					},
					podIdentityAssociations: map[string][]podIdentityAssociation{
						"my-eks-cluster": {
							{ClusterName: "my-eks-cluster", ServiceAccountName: "sa1", RoleArn: "arn:aws:iam::123456789012:role/sa1-role"},
							{ClusterName: "my-eks-cluster", ServiceAccountName: "sa2", RoleArn: "arn:aws:iam::123456789012:role/sa2-role"},
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
	clusters map[string][]mockCluster // region -> clusters
}

type mockCluster struct {
	clusterName             string
	serviceAccounts         map[string][]serviceAccount         // cluster name -> service accounts
	podIdentityAssociations map[string][]podIdentityAssociation // cluster name -> pod identity associations
}

func (c *mockClient) listEKSClusters(roleArn, region string) ([]string, error) {
	clusters := c.clusters[region]
	var clusterNames []string
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.clusterName)
	}
	return clusterNames, nil
}

func (c *mockClient) listServiceAccounts(roleArn, region, clusterName string) ([]serviceAccount, error) {
	for _, cluster := range c.clusters[region] {
		if cluster.clusterName == clusterName {
			return cluster.serviceAccounts[clusterName], nil
		}
	}
	return nil, errors.New("cluster not found")
}

func (c *mockClient) listPodIdentityAssociations(roleArn, region, clusterName string) ([]podIdentityAssociation, error) {
	for _, cluster := range c.clusters[region] {
		if cluster.clusterName == clusterName {
			return cluster.podIdentityAssociations[clusterName], nil
		}
	}
	return nil, errors.New("cluster not found")
}
