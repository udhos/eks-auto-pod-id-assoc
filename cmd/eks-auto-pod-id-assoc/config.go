package main

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

type config struct {
	Clusters []configCluster `yaml:"clusters"`
}

type configCluster struct {
	RoleArn     string `yaml:"role_arn"`
	Region      string `yaml:"region"`
	ClusterName string `yaml:"cluster_name"`
	Self        bool   `yaml:"self"`
}

func loadConfigFromFile(input string) (config, error) {
	data, errRead := os.ReadFile(input)
	if errRead != nil {
		// get current dir
		cwd, err := os.Getwd()
		if err != nil {
			errorf("failed to get current dir: %v", err)
		}
		return config{}, fmt.Errorf("failed to read config file %s in directory %s: %w", input, cwd, errRead)
	}

	return loadConfig(data)
}

func loadConfig(data []byte) (config, error) {
	var cfg config
	err := yaml.Unmarshal(data, &cfg)
	return cfg, err
}
