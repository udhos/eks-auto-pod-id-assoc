package main

import (
	"testing"

	"github.com/udhos/boilerplate/envconfig"
)

/*
clusters:
  # use a specific role to access a specific cluster
  - role_arn: "arn:aws:iam::123456789012:role/AnotherRole"
    region: "us-east-1"
    cluster_name: "example-cluster-2"

  # use default role and discover all clusters in the region
  - region: "sa-east-1"
*/

func TestApp(t *testing.T) {

	env := envconfig.NewSimple("TestApp")
	configFile := env.String("CONFIG_FILE", "../../config.yaml")
	cfg, err := loadConfig(configFile)
	if err != nil {
		fatalf("failed to load config: %s: %v", configFile, err)
	}

	client := newMockClient()

	app := newApplication(cfg, client)
	app.run()
}
