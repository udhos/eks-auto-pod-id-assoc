package main

import (
	"errors"
	"fmt"
	"regexp"

	"go.yaml.in/yaml/v4"
)

type application struct {
	cfg    config
	client clientInterface
}

func newApplication(cfg config, awsSource clientInterface) *application {
	app := &application{
		cfg:    cfg,
		client: awsSource,
	}
	return app
}

type cluster struct {
	Config                  configCluster
	ServiceAccounts         []serviceAccount
	PodIdentityAssociations []podIdentityAssociation
}

func (a *application) run() {
	clusterList := a.discoverClusters()

	// marshal to YAML and print to stdout
	infof("discovered cluster list:")
	yamlBytes, err := yaml.Marshal(clusterList)
	if err != nil {
		fatalf("failed to marshal cluster list to YAML: %v", err)
	}
	fmt.Println(string(yamlBytes))

}

func (a *application) discoverClusters() []cluster {
	//
	// discover clusters, service accounts and pod identity associations
	//

	var clusterList []cluster
	for _, c := range a.cfg.Clusters {

		var clusterNames []string

		//
		// discover all clusters in the region
		//
		names, err := a.client.listEKSClusters(c.RoleArn, c.Region)
		if err != nil {
			fatalf("failed to list EKS clusters in region %s: %v", c.Region, err)
		}

		// compile pattern
		pattern, errPattern := regexp.Compile(c.ClusterName)
		if errPattern != nil {
			fatalf("failed to compile cluster name pattern %s: %v", c.ClusterName, errPattern)
		}

		// pick only matching patterns
		for _, name := range names {
			if !pattern.MatchString(name) {
				infof("skipping cluster %s in region %s: does not match pattern %s", name, c.Region, c.ClusterName)
				continue
			}
			clusterNames = append(clusterNames, name)
		}

		// discover service accounts and pod identity associations for each cluster
		for _, clusterName := range clusterNames {
			saList, err := a.client.listServiceAccounts(c.RoleArn, c.Region, clusterName)
			if err != nil {
				fatalf("failed to list service accounts for cluster %s: %v", clusterName, err)
			}

			piList, err := a.client.listPodIdentityAssociations(c.RoleArn, c.Region, clusterName)
			if err != nil {
				fatalf("failed to list pod identity associations for cluster %s: %v", clusterName, err)
			}

			cc := c
			cc.ClusterName = clusterName // discovered cluster name

			clusterList = append(clusterList, cluster{
				Config:                  cc,
				ServiceAccounts:         saList,
				PodIdentityAssociations: piList,
			})
		}
	}

	return clusterList
}

type clientInterface interface {
	listEKSClusters(roleArn, region string) ([]string, error)
	listServiceAccounts(roleArn, region, clusterName string) ([]serviceAccount, error)
	listPodIdentityAssociations(roleArn, region, clusterName string) ([]podIdentityAssociation, error)
	createPodIdentityAssociation(roleArn, region, clusterName, serviceAccountName, serviceAccountRoleArn string) error
	deletePodIdentityAssociation(roleArn, region, clusterName, associationID string) error
}

type serviceAccount struct {
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace"`
	AwsRoleArn string `yaml:"aws_role_arn"`
}

type podIdentityAssociation struct {
	AssociationID      string `yaml:"association_id"`
	ClusterName        string `yaml:"cluster_name"`
	ServiceAccountName string `yaml:"service_account_name"`
	RoleArn            string `yaml:"role_arn"`
}

type realClient struct{}

func (c *realClient) listEKSClusters(roleArn, region string) ([]string, error) {
	return []string{}, errors.New("not implemented")
}

func (c *realClient) listServiceAccounts(roleArn, region, clusterName string) ([]serviceAccount, error) {
	return []serviceAccount{}, errors.New("not implemented")
}

func (c *realClient) listPodIdentityAssociations(roleArn, region, clusterName string) ([]podIdentityAssociation, error) {
	return []podIdentityAssociation{}, errors.New("not implemented")
}

func (c *realClient) createPodIdentityAssociation(roleArn, region, clusterName, serviceAccountName, serviceAccountRoleArn string) error {
	return errors.New("not implemented")
}

func (c *realClient) deletePodIdentityAssociation(roleArn, region, clusterName, associationID string) error {
	return errors.New("not implemented")
}
