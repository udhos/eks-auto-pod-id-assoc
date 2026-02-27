package main

import (
	"testing"

	"github.com/udhos/boilerplate/envconfig"
)

func TestApp(t *testing.T) {

	env := envconfig.NewSimple("TestApp")
	configFile := env.String("CONFIG_FILE", "../../config.yaml")
	cfg, err := loadConfigFromFile(configFile)
	if err != nil {
		t.Fatalf("failed to load config: %s: %v", configFile, err)
	}

	client := newMockClient()

	app := newApplication(cfg, client)
	app.run()
}

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
