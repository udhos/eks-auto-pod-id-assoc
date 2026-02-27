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
}

func loadConfig(input string) (config, error) {
	data, errRead := os.ReadFile(input)
	if errRead != nil {
		// get current dir
		cwd, err := os.Getwd()
		if err != nil {
			errorf("failed to get current dir: %v", err)
		}
		return config{}, fmt.Errorf("failed to read config file %s in directory %s: %w", input, cwd, errRead)
	}

	var cfg config
	err := yaml.Unmarshal(data, &cfg)
	if err != nil {
		return config{}, err
	}

	return cfg, nil
}
