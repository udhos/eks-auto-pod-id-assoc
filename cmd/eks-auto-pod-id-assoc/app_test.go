package main

import (
	"errors"
	"testing"

	"github.com/segmentio/ksuid"
)

func TestDiscoveryRegion(t *testing.T) {

	const conf = `
clusters:
  - region: us-east-1
`

	cfg, err := loadConfig([]byte(conf))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := newMockClient()

	// count loaded clusters
	var countClusters int
	for _, clusters := range client.regions {
		countClusters += len(clusters)
	}
	if countClusters <= 1 {
		t.Fatalf("total_clusters=%d should be greater than 1",
			countClusters)
	}

	app := newApplication(cfg, client)

	clusterList := app.discoverClusters()

	foundClusters := len(clusterList)

	if foundClusters != 1 {
		t.Fatalf("total_clusters=%d discovered_clusters=%d (expecting 1 at region us-east-1)",
			countClusters, foundClusters)
	}

	t.Logf("total_clusters=%d discovered_clusters=%d (region us-east-1)",
		countClusters, foundClusters)
}

func TestDiscoveryClusterNameRegex(t *testing.T) {

	const conf = `
clusters:
  - region: sa-east-1
    cluster_name: ^my-
`

	cfg, err := loadConfig([]byte(conf))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := newMockClient()

	// count loaded clusters
	countClusters := len(client.regions["sa-east-1"])
	if countClusters <= 1 {
		t.Fatalf("total_clusters=%d should be greater than 1 at region sa-east-1",
			countClusters)
	}

	app := newApplication(cfg, client)

	clusterList := app.discoverClusters()

	foundClusters := len(clusterList)

	if foundClusters != 1 {
		t.Fatalf("total_clusters=%d discovered_clusters=%d (expecting 1 at region sa-east-1)",
			countClusters, foundClusters)
	}

	t.Logf("total_clusters=%d discovered_clusters=%d (region sa-east-1)",
		countClusters, foundClusters)
}

func TestAddServiceAccount(t *testing.T) {

	const conf = `
clusters:
  - region: us-east-1
    cluster_name: ^example-cluster-2$
`

	cfg, err := loadConfig([]byte(conf))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := newMockClient()

	app := newApplication(cfg, client)

	{
		clusterList := app.discoverClusters()
		foundClusters := len(clusterList)
		if foundClusters != 1 {
			t.Fatalf("found %d clusters, expected 1", foundClusters)
		}
		cl := clusterList[0]
		countServiceAccounts := len(cl.ServiceAccounts)
		if countServiceAccounts != 1 {
			t.Fatalf("found %d service accounts, expected 1", countServiceAccounts)
		}
		countPIAs := len(cl.PodIdentityAssociations)
		if countPIAs != 1 {
			t.Fatalf("found %d PIAs, expected 1", countPIAs)
		}
	}

	app.run()

	{
		clusterList := app.discoverClusters() // reload
		cl := clusterList[0]
		countServiceAccounts := len(cl.ServiceAccounts)
		if countServiceAccounts != 1 {
			t.Fatalf("after run 1: found %d service accounts, expected 1", countServiceAccounts)
		}
		countPIAs := len(cl.PodIdentityAssociations)
		if countPIAs != 1 {
			t.Fatalf("after run 1: found %d PIAs, expected 1", countPIAs)
		}
	}

	// add SA
	{
		cl := client.regions["us-east-1"][0]
		sa := serviceAccount{
			Name:       "newSa",
			Namespace:  "default",
			AwsRoleArn: "newAwsRoleARN",
		}
		cl.serviceAccounts = append(cl.serviceAccounts, sa)
		client.regions["us-east-1"][0] = cl
	}

	{
		clusterList := app.discoverClusters() // reload
		cl := clusterList[0]
		countServiceAccounts := len(cl.ServiceAccounts)
		if countServiceAccounts != 2 {
			t.Fatalf("after SA: found %d service accounts, expected 2", countServiceAccounts)
		}
		countPIAs := len(cl.PodIdentityAssociations)
		if countPIAs != 1 {
			t.Fatalf("after SA: found %d PIAs, expected 1", countPIAs)
		}
	}

	app.run()

	{
		clusterList := app.discoverClusters() // reload
		cl := clusterList[0]
		countServiceAccounts := len(cl.ServiceAccounts)
		if countServiceAccounts != 2 {
			t.Fatalf("after run 2: found %d service accounts, expected 2", countServiceAccounts)
		}
		countPIAs := len(cl.PodIdentityAssociations)
		if countPIAs != 2 {
			t.Fatalf("after run 2: found %d PIAs, expected 2", countPIAs)
		}
	}

}

func TestRemoveServiceAccount(t *testing.T) {

	const conf = `
clusters:
  - region: us-east-1
    cluster_name: ^example-cluster-2$
`

	cfg, err := loadConfig([]byte(conf))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := newMockClient()

	app := newApplication(cfg, client)

	{
		clusterList := app.discoverClusters()
		foundClusters := len(clusterList)
		if foundClusters != 1 {
			t.Fatalf("found %d clusters, expected 1", foundClusters)
		}
		cl := clusterList[0]
		countServiceAccounts := len(cl.ServiceAccounts)
		if countServiceAccounts != 1 {
			t.Fatalf("found %d service accounts, expected 1", countServiceAccounts)
		}
		countPIAs := len(cl.PodIdentityAssociations)
		if countPIAs != 1 {
			t.Fatalf("found %d PIAs, expected 1", countPIAs)
		}
	}

	app.run()

	{
		clusterList := app.discoverClusters() // reload
		cl := clusterList[0]
		countServiceAccounts := len(cl.ServiceAccounts)
		if countServiceAccounts != 1 {
			t.Fatalf("after run 1: found %d service accounts, expected 1", countServiceAccounts)
		}
		countPIAs := len(cl.PodIdentityAssociations)
		if countPIAs != 1 {
			t.Fatalf("after run 1: found %d PIAs, expected 1", countPIAs)
		}
	}

	// remove SA
	{
		cl := client.regions["us-east-1"][0]
		cl.serviceAccounts = nil
		client.regions["us-east-1"][0] = cl
	}

	{
		clusterList := app.discoverClusters() // reload
		cl := clusterList[0]
		countServiceAccounts := len(cl.ServiceAccounts)
		if countServiceAccounts != 0 {
			t.Fatalf("after SA: found %d service accounts, expected 0", countServiceAccounts)
		}
		countPIAs := len(cl.PodIdentityAssociations)
		if countPIAs != 1 {
			t.Fatalf("after SA: found %d PIAs, expected 1", countPIAs)
		}
	}

	app.run()

	{
		clusterList := app.discoverClusters() // reload
		cl := clusterList[0]
		countServiceAccounts := len(cl.ServiceAccounts)
		if countServiceAccounts != 0 {
			t.Fatalf("after run 2: found %d service accounts, expected 0", countServiceAccounts)
		}
		countPIAs := len(cl.PodIdentityAssociations)
		if countPIAs != 0 {
			t.Fatalf("after run 2: found %d PIAs, expected 0", countPIAs)
		}
	}

}

func TestServiceAccountExcludeNamespace(t *testing.T) {
	saList := []serviceAccount{
		{
			Namespace: "default",
		},
		{
			Namespace: "kube-system",
		},
	}

	result := serviceAccountsExcludeNamespace(saList, []string{"kube-system", "other"})

	if len(result) != 1 {
		t.Errorf("unexpected list size:%d", len(result))
	}

	if result[0].Namespace != "default" {
		t.Errorf("unexpected namespace: %s", result[0].Namespace)
	}
}

func TestPodIdentityAssociationExcludeNamespace(t *testing.T) {
	pia := []podIdentityAssociation{
		{
			ServiceAccountNamespace: "default",
		},
		{
			ServiceAccountNamespace: "kube-system",
		},
	}

	result := podIdentityAssociationExcludeNamespace(pia, []string{"kube-system", "other"})

	if len(result) != 1 {
		t.Errorf("unexpected list size:%d", len(result))
	}

	if result[0].ServiceAccountNamespace != "default" {
		t.Errorf("unexpected namespace: %s", result[0].ServiceAccountNamespace)
	}
}

func newMockClient() *mockClient {
	client := &mockClient{
		regions: map[string][]mockCluster{
			"self": {
				{
					clusterName: "my-cluster",
					serviceAccounts: []serviceAccount{
						{Name: "sa1", Namespace: "default", AwsRoleArn: "arn:aws:iam::123456789012:role/sa1-role"},
					},
					podIdentityAssociations: []podIdentityAssociation{
						{
							AssociationID:           "example-assoc-id-1",
							ClusterName:             "my-cluster",
							ServiceAccountNamespace: "default",
							ServiceAccountName:      "sa1",
							RoleArn:                 "arn:aws:iam::123456789012:role/sa1-role",
						},
					},
				},
			},
			"us-east-1": {
				{
					clusterName: "example-cluster-2",
					serviceAccounts: []serviceAccount{
						{Name: "sa1", Namespace: "default", AwsRoleArn: "arn:aws:iam::123456789012:role/sa1-role"},
					},
					podIdentityAssociations: []podIdentityAssociation{
						{
							AssociationID:           "example-assoc-id-1",
							ClusterName:             "example-cluster-2",
							ServiceAccountNamespace: "default",
							ServiceAccountName:      "sa1",
							RoleArn:                 "arn:aws:iam::123456789012:role/sa1-role",
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
							AssociationID:           "example-assoc-id-1",
							ClusterName:             "my-eks-cluster",
							ServiceAccountNamespace: "default",
							ServiceAccountName:      "sa1",
							RoleArn:                 "arn:aws:iam::123456789012:role/sa1-role",
						},
						{
							AssociationID:           "example-assoc-id-2",
							ClusterName:             "my-eks-cluster",
							ServiceAccountNamespace: "kube-system",
							ServiceAccountName:      "sa2",
							RoleArn:                 "arn:aws:iam::123456789012:role/sa2-role",
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

func (c *mockClient) listEKSClusters(_, region string) ([]string, error) {
	clusters := c.regions[region]
	var clusterNames []string
	for _, cluster := range clusters {
		clusterNames = append(clusterNames, cluster.clusterName)
	}
	return clusterNames, nil
}

func (c *mockClient) listServiceAccounts(self bool, _, region,
	clusterName, _ string) ([]serviceAccount, error) {
	if self {
		region = "self"
	}
	for _, cluster := range c.regions[region] {
		if cluster.clusterName == clusterName {
			return cluster.serviceAccounts, nil
		}
	}
	return nil, errors.New("cluster not found")
}

func (c *mockClient) listPodIdentityAssociations(self bool, _, region,
	clusterName string) ([]podIdentityAssociation, error) {
	if self {
		region = "self"
	}
	for _, cluster := range c.regions[region] {
		if cluster.clusterName == clusterName {
			return cluster.podIdentityAssociations, nil
		}
	}
	return nil, errors.New("cluster not found")
}

func (c *mockClient) createPodIdentityAssociation(self bool, _, region,
	clusterName, serviceAccountName, serviceAccountRoleArn string) error {

	if self {
		region = "self"
	}

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

func (c *mockClient) deletePodIdentityAssociation(self bool, _, region,
	clusterName, associationID string) error {

	if self {
		region = "self"
	}

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
