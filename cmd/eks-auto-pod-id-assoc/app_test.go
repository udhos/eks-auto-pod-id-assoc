package main

import (
	"errors"
	"testing"

	"github.com/segmentio/ksuid"
	"k8s.io/client-go/kubernetes"
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

	const namespace = ""

	met := newMetrics(namespace, defaultLatencyBucketsSeconds)

	app := newApplication(cfg, met, client)

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

	const namespace = ""

	met := newMetrics(namespace, defaultLatencyBucketsSeconds)

	app := newApplication(cfg, met, client)

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

	const namespace = ""

	met := newMetrics(namespace, defaultLatencyBucketsSeconds)

	app := newApplication(cfg, met, client)

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

	const namespace = ""

	met := newMetrics(namespace, defaultLatencyBucketsSeconds)

	app := newApplication(cfg, met, client)

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

func TestServiceAccountExcludeServiceAccounts(t *testing.T) {
	saList := []serviceAccount{
		{
			Name:      "sa1",
			Namespace: "default",
		},
		{
			Name:      "sa2",
			Namespace: "ns1",
		},
		{
			Name:      "sa2",
			Namespace: "ns2",
		},
		{
			Name: "sa3",
		},
	}

	exclude := []matchServiceAccount{
		{
			Name: "^sa1$",
		},
		{
			Namespace: "^ns1$",
		},
		{
			Name:      "^sa2$",
			Namespace: "^ns2$",
		},
	}

	if errCompile := compileServiceAccountList(exclude); errCompile != nil {
		t.Fatalf("compile: %v", errCompile)
	}

	result := serviceAccountsExcludeServiceAccounts(saList, exclude)

	if len(result) != 1 {
		t.Errorf("unexpected list size:%d", len(result))
	}

	if result[0].Name != "sa3" {
		t.Errorf("unexpected namespace: %s", result[0].Namespace)
	}
}

func TestServiceAccountExcludePodIdentityAssociations(t *testing.T) {
	saList := []podIdentityAssociation{
		{
			ServiceAccountName:      "sa1",
			ServiceAccountNamespace: "default",
		},
		{
			ServiceAccountName:      "sa2",
			ServiceAccountNamespace: "ns1",
		},
		{
			ServiceAccountName:      "sa2",
			ServiceAccountNamespace: "ns2",
		},
		{
			ServiceAccountName: "sa3",
		},
	}

	exclude := []matchServiceAccount{
		{
			Name: "^sa1$",
		},
		{
			Namespace: "^ns1$",
		},
		{
			Name:      "^sa2$",
			Namespace: "^ns2$",
		},
	}

	if errCompile := compileServiceAccountList(exclude); errCompile != nil {
		t.Fatalf("compile: %v", errCompile)
	}

	result := podIdentityAssociationExcludeServiceAccounts(saList, exclude)

	if len(result) != 1 {
		t.Errorf("unexpected list size:%d", len(result))
	}

	if result[0].ServiceAccountName != "sa3" {
		t.Errorf("unexpected namespace: %s", result[0].ServiceAccountName)
	}
}

func TestRestrictedRoles(t *testing.T) {

	const input = `
clusters:
  - restrict_roles:
      - role_arn: ^arn:aws:iam::123456789012:role/role1$ # only this role is restricted
        allow:
          # only this SA can use the role
          - name: ^sa3$
            namespace: ^ns3$
`

	cfg, err := loadConfig([]byte(input))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}

	if len(cfg.Clusters) != 1 {
		t.Fatalf("wrong number of clusters: %d", len(cfg.Clusters))
	}

	serviceAccounts := []serviceAccount{
		{
			Name:       "sa1",
			Namespace:  "ns1",
			AwsRoleArn: "role1", // this role is not restricted
		},
		{
			Name:       "sa2",
			Namespace:  "ns2",
			AwsRoleArn: "arn:aws:iam::123456789012:role/role1", // this role is restricted
		},
		{
			Name:       "sa3",
			Namespace:  "ns3",
			AwsRoleArn: "arn:aws:iam::123456789012:role/role1", // this role is restricted but the SA is allowed
		},
	}

	result := excludeRestrictedRoles(serviceAccounts, cfg.Clusters[0].RestrictRoles)

	if len(result) != 2 {
		t.Fatalf("wrong number of service accounts: %d", len(result))
	}

	if result[0].Name != "sa1" {
		t.Fatalf("wrong SA 0 name: %s", result[0].Name)
	}
	if result[0].Namespace != "ns1" {
		t.Fatalf("wrong SA 0 namespace: %s", result[0].Namespace)
	}
	if result[1].Name != "sa3" {
		t.Fatalf("wrong SA 1 name: %s", result[1].Name)
	}
	if result[1].Namespace != "ns3" {
		t.Fatalf("wrong SA 1 namespace: %s", result[1].Namespace)
	}
}

func TestRestrictedRolesOrderCatchAllAtEnd(t *testing.T) {

	const input = `
clusters:
  - restrict_roles:
      - role_arn: ^role1$
        allow:
          - name: ^sa1$
            namespace: ^ns1$
      - role_arn: ""
        allow:
          - name: ^sa2$
            namespace: ^ns2$
`

	cfg, err := loadConfig([]byte(input))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}

	if len(cfg.Clusters) != 1 {
		t.Fatalf("wrong number of clusters: %d", len(cfg.Clusters))
	}

	serviceAccounts := []serviceAccount{
		{
			// allowed
			Name:       "sa1",
			Namespace:  "ns1",
			AwsRoleArn: "role1",
		},
		{
			// not allowed
			Name:       "sa2",
			Namespace:  "ns2",
			AwsRoleArn: "role1",
		},
		{
			// not allowed
			Name:       "xxx",
			Namespace:  "xxx",
			AwsRoleArn: "role1",
		},
		{
			// allowed
			Name:       "sa2",
			Namespace:  "ns2",
			AwsRoleArn: "role2",
		},
		{
			// not allowed
			Name:       "xxx",
			Namespace:  "xxx",
			AwsRoleArn: "role2",
		},
	}

	result := excludeRestrictedRoles(serviceAccounts, cfg.Clusters[0].RestrictRoles)

	if len(result) != 2 {
		t.Fatalf("wrong number of service accounts: %d", len(result))
	}

	if result[0].Name != "sa1" {
		t.Fatalf("wrong SA 0 name: %s", result[0].Name)
	}
	if result[0].Namespace != "ns1" {
		t.Fatalf("wrong SA 0 namespace: %s", result[0].Namespace)
	}
	if result[1].Name != "sa2" {
		t.Fatalf("wrong SA 1 name: %s", result[1].Name)
	}
	if result[1].Namespace != "ns2" {
		t.Fatalf("wrong SA 1 namespace: %s", result[1].Namespace)
	}
}

func TestRestrictedRolesOrderCatchAllAtBegin(t *testing.T) {

	const input = `
clusters:
  - restrict_roles:
      - role_arn: ""
        allow:
          - name: ^sa2$
            namespace: ^ns2$
      - role_arn: ^role1$
        allow:
          - name: ^sa1$
            namespace: ^ns1$
`

	cfg, err := loadConfig([]byte(input))
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}

	if len(cfg.Clusters) != 1 {
		t.Fatalf("wrong number of clusters: %d", len(cfg.Clusters))
	}

	serviceAccounts := []serviceAccount{
		{
			// not allowed
			Name:       "sa1",
			Namespace:  "ns1",
			AwsRoleArn: "role1",
		},
		{
			// allowed
			Name:       "sa2",
			Namespace:  "ns2",
			AwsRoleArn: "role1",
		},
		{
			// not allowed
			Name:       "xxx",
			Namespace:  "xxx",
			AwsRoleArn: "role1",
		},
		{
			// allowed
			Name:       "sa2",
			Namespace:  "ns2",
			AwsRoleArn: "role2",
		},
		{
			// not allowed
			Name:       "xxx",
			Namespace:  "xxx",
			AwsRoleArn: "role2",
		},
	}

	result := excludeRestrictedRoles(serviceAccounts, cfg.Clusters[0].RestrictRoles)

	if len(result) != 2 {
		t.Fatalf("wrong number of service accounts: %d", len(result))
	}

	if result[0].Name != "sa2" {
		t.Fatalf("wrong SA 0 name: %s", result[0].Name)
	}
	if result[0].Namespace != "ns2" {
		t.Fatalf("wrong SA 0 namespace: %s", result[0].Namespace)
	}
	if result[1].Name != "sa2" {
		t.Fatalf("wrong SA 1 name: %s", result[1].Name)
	}
	if result[1].Namespace != "ns2" {
		t.Fatalf("wrong SA 1 namespace: %s", result[1].Namespace)
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
						},
						{
							AssociationID:           "example-assoc-id-2",
							ClusterName:             "my-eks-cluster",
							ServiceAccountNamespace: "kube-system",
							ServiceAccountName:      "sa2",
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

func (c *mockClient) getKubeClient(_ bool, _,
	_, _ string) (*kubernetes.Clientset, error) {
	return nil, errors.New("mockClient.getKubeClient: NOT IMPLEMENTED")
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
	clusterName, _, serviceAccountName,
	_ string, _ map[string]string) error {

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
