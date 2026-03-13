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
	RoleArn                            string                `yaml:"role_arn"`
	Region                             string                `yaml:"region"`
	ClusterName                        string                `yaml:"cluster_name"`
	Self                               bool                  `yaml:"self"`
	Annotation                         string                `yaml:"annotation"`
	ExcludeServiceAccounts             []matchServiceAccount `yaml:"exclude_service_accounts"`
	RestrictRoles                      []restrictRole        `yaml:"restrict_roles"`
	PodIdentityAssociationTags         map[string]string     `yaml:"pod_identity_association_tags"`
	MaxConcurrency                     int                   `yaml:"max_concurrency"`
	PurgeExternalStaleAssociations     bool                  `yaml:"purge_external_stale_associations"`
	ForceIterativeAssociationDiscovery bool                  `yaml:"force_iterative_association_discovery"`
}

const defaultMaxConcurrency = 5

type restrictRole struct {
	RoleArn string                `yaml:"role_arn"`
	Allow   []matchServiceAccount `yaml:"allow"`

	matchRole pattern
}

func (rr *restrictRole) compile() error {
	for i, sa := range rr.Allow {
		if errSA := sa.compile(); errSA != nil {
			return errSA
		}
		rr.Allow[i] = sa // write back
	}

	matchRole, errRole := newPattern(rr.RoleArn)
	if errRole != nil {
		return errRole
	}
	rr.matchRole = matchRole
	return nil
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
		if errCompile := compileServiceAccountList(cl.ExcludeServiceAccounts); errCompile != nil {
			return config{}, fmt.Errorf("exclude_service_accounts compile error: cluster=%q: %w", cl.ClusterName, errCompile)
		}

		if errCompile := compileRestrictRolesList(cl.RestrictRoles); errCompile != nil {
			return config{}, fmt.Errorf("restrict_roles compile error: cluster=%q: %w", cl.ClusterName, errCompile)
		}

		cfg.Clusters[c] = cl // write back modified cluster
	}

	for _, cl := range cfg.Clusters {

		// check exclude service accounts
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

		// check restricted roles
		for _, restRol := range cl.RestrictRoles {

			if restRol.matchRole.re == nil {
				return config{}, fmt.Errorf("role_arn regex is nil: restrict_roles: cluster=%q role_arn=%q",
					cl.ClusterName, restRol.RoleArn)
			}

			for _, srvAcc := range restRol.Allow {
				if srvAcc.matchName.re == nil {
					return config{}, fmt.Errorf("name regex is nil: restrict_roles: cluster=%q role_arn=%q allow sa_name=%q",
						cl.ClusterName, restRol.RoleArn, srvAcc.Name)
				}
				if srvAcc.matchNamespace.re == nil {
					return config{}, fmt.Errorf("namespace regex is nil: restrict_roles: cluster=%q role_arn=%q allow sa_namespace=%q",
						cl.ClusterName, restRol.RoleArn, srvAcc.Namespace)
				}
			}
		}
	}

	cfg = defaultConfig(cfg)

	return cfg, err
}

func defaultConfig(cfg config) config {
	for c, cl := range cfg.Clusters {

		// add default tags to cluster
		if len(cl.PodIdentityAssociationTags) == 0 {
			cl.PodIdentityAssociationTags = defaultTags
			cfg.Clusters[c] = cl // write back
		}
	}
	return cfg
}

func compileRestrictRolesList(list []restrictRole) error {
	for rr, restRol := range list {
		if errCompile := restRol.compile(); errCompile != nil {
			return fmt.Errorf("compile error: restrict_roles: index=%d role_arn=%q: %w",
				rr, restRol.RoleArn, errCompile)
		}
		list[rr] = restRol // write back modified restrict role
	}
	return nil
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
