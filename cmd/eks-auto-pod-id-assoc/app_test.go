package main

import (
	"testing"
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
