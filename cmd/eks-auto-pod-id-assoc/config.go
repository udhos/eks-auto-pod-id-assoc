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
	RoleArn                string                `yaml:"role_arn"`
	Region                 string                `yaml:"region"`
	ClusterName            string                `yaml:"cluster_name"`
	Self                   bool                  `yaml:"self"`
	Annotation             string                `yaml:"annotation"`
	ExcludeServiceAccounts []matchServiceAccount `yaml:"exclude_service_accounts"`
	RestrictRoles          map[string][]string   `yaml:"restrict_roles"`
}

type matchServiceAccount struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`

	matchName      pattern
	matchNamespace pattern
}

func (m *matchServiceAccount) compile() error {
	matchName, errName := newPattern(m.Name)
	if errName != nil {
		return errName
	}
	matchNamespace, errNamespace := newPattern(m.Namespace)
	if errNamespace != nil {
		return errNamespace
	}
	m.matchName = matchName
	m.matchNamespace = matchNamespace
	return nil
}

func (m *matchServiceAccount) match(name, namespace string) bool {
	return m.matchName.match(name) && m.matchNamespace.match(namespace)
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
	if err != nil {
		return config{}, err
	}

	for c, cl := range cfg.Clusters {
		errCompile := compileServiceAccountList(cl.ExcludeServiceAccounts)
		if errCompile != nil {
			return config{}, fmt.Errorf("exclude_service_accounts compile error: cluster=%q: %w", cl.ClusterName, errCompile)
		}
		cfg.Clusters[c] = cl // write back modified cluster
	}

	for _, cl := range cfg.Clusters {
		for ex, excSa := range cl.ExcludeServiceAccounts {
			if excSa.matchName.re == nil {
				return config{}, fmt.Errorf("name regex is nil: exclude_service_accounts: cluster=%q index=%d name=%q",
					cl.ClusterName, ex, excSa.Name)
			}
			if excSa.matchNamespace.re == nil {
				return config{}, fmt.Errorf("namespace regex is nil: exclude_service_accounts: cluster=%q index=%d namespace=%q",
					cl.ClusterName, ex, excSa.Namespace)
			}
		}
	}

	return cfg, err
}

func compileServiceAccountList(list []matchServiceAccount) error {
	for ex, excSa := range list {
		if errCompile := excSa.compile(); errCompile != nil {
			return fmt.Errorf("compile error: exclude_service_accounts: index=%d name=%q namespace=%q: %w",
				ex, excSa.Name, excSa.Namespace, errCompile)
		}
		list[ex] = excSa // write back modified sa
	}
	return nil
}
