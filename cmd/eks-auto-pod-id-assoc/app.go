package main

import (
	"fmt"
	"regexp"

	"go.yaml.in/yaml/v4"
)

type application struct {
	cfg    config
	client clientInterface
}

func newApplication(cfg config, client clientInterface) *application {
	app := &application{
		cfg:    cfg,
		client: client,
	}
	return app
}

type cluster struct {
	Config                  configCluster            `yaml:"config"`
	ServiceAccounts         []serviceAccount         `yaml:"service_accounts"`
	PodIdentityAssociations []podIdentityAssociation `yaml:"pod_identity_associations"`
}

func (a *application) run() {
	clusterList := a.discoverClusters()

	dumpClusters(clusterList, "discovered clusters:")

	a.reconcileClusters(clusterList)

	dumpClusters(clusterList, "reconciled clusters:")
}

func dumpClusters(clusterList []cluster, label string) {
	fmt.Println(label)
	yamlBytes, _ := yaml.Marshal(clusterList)
	fmt.Println(string(yamlBytes))
}

func (a *application) reconcileClusters(clusterList []cluster) {
	for _, cl := range clusterList {

		clusterLabel := fmt.Sprintf("role=%q region=%q cluster=%q",
			cl.Config.RoleArn, cl.Config.Region, cl.Config.ClusterName)

		// create associations for service accounts without associations
		{
			missingServiceAccounts := a.findMissingServiceAccounts(cl)

			infof("%s found missing service accounts: %d",
				clusterLabel, len(missingServiceAccounts))

			for i, sa := range missingServiceAccounts {
				label := fmt.Sprintf("%d/%d", i+1, len(missingServiceAccounts))
				if err := a.client.createPodIdentityAssociation(cl.Config.Self, cl.Config.RoleArn,
					cl.Config.Region, cl.Config.ClusterName, sa.Name, sa.AwsRoleArn); err != nil {
					errorf("%s failure creating pod identity association %s: serviceAccount=%q serviceAccountRoleArn=%q: %v",
						clusterLabel, label, sa.Name, sa.AwsRoleArn, err)
					continue
				}
				infof("%s created pod identity association %s: serviceAccount=%q serviceAccountRoleArn=%q",
					clusterLabel, label, sa.Name, sa.AwsRoleArn)
			}
		}

		// delete associations for service accounts that don't exist
		{
			stalePIAs := a.findStalePodIdentityAssociations(cl)

			infof("%s found stale pod identity associations: %d",
				clusterLabel, len(stalePIAs))

			for i, pia := range stalePIAs {
				label := fmt.Sprintf("%d/%d", i+1, len(stalePIAs))
				if err := a.client.deletePodIdentityAssociation(cl.Config.Self, cl.Config.RoleArn,
					cl.Config.Region, pia.ClusterName, pia.AssociationID); err != nil {
					errorf("%s failure deleting pod identity association %s: associationID=%q serviceAccount=%q: %v",
						clusterLabel, label, pia.AssociationID, pia.ServiceAccountName, err)
					continue
				}
				infof("%s deleted pod identity association %s: associationID=%q serviceAccount=%q",
					clusterLabel, label, pia.AssociationID, pia.ServiceAccountName)
			}
		}
	}
}

func (a *application) findMissingServiceAccounts(cl cluster) []serviceAccount {

	var missing []serviceAccount

	for _, sa := range cl.ServiceAccounts {
		var found bool

		for _, pia := range cl.PodIdentityAssociations {
			if sa.Name == pia.ServiceAccountName &&
				sa.Namespace == pia.ServiceAccountNamespace {
				found = true // found PIA for SA
				break
			}
		}

		if !found {
			missing = append(missing, sa) // add SA without PIA as missing
		}
	}

	return missing
}

func (a *application) findStalePodIdentityAssociations(cl cluster) []podIdentityAssociation {

	var stale []podIdentityAssociation

	for _, pia := range cl.PodIdentityAssociations {
		var found bool

		for _, sa := range cl.ServiceAccounts {
			if sa.Name == pia.ServiceAccountName &&
				sa.Namespace == pia.ServiceAccountNamespace {
				found = true // found SA for PIA
				break
			}
		}

		if !found {
			stale = append(stale, pia) // add PIA without SA as stale
		}
	}

	return stale
}

func (a *application) findClusterNames(c configCluster) ([]string, error) {

	if c.Self {
		// we cannot discover our name, we need it specified exactly.
		return []string{c.ClusterName}, nil
	}

	var clusterNames []string

	// compile pattern
	pattern, errPattern := regexp.Compile(c.ClusterName)
	if errPattern != nil {
		return nil, fmt.Errorf("failed to compile cluster name pattern %s: %w",
			c.ClusterName, errPattern)
	}

	// list clusters in the region
	names, err := a.client.listEKSClusters(c.RoleArn, c.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to list EKS clusters in region %s: %w",
			c.Region, err)
	}

	// pick only matching patterns
	for _, name := range names {
		match := pattern.MatchString(name)
		infof("region=%s pattern=%q cluster_name=%s matched=%t",
			c.Region, c.ClusterName, name, match)
		if !match {
			continue // skip this cluster name
		}
		clusterNames = append(clusterNames, name)
	}

	return clusterNames, nil
}

func (a *application) discoverClusters() []cluster {
	//
	// discover clusters, service accounts and pod identity associations
	//

	var clusterList []cluster
	for _, c := range a.cfg.Clusters {

		clusterNames, errNames := a.findClusterNames(c)
		if errNames != nil {
			errorf("%v", errNames)
			continue // skip this cluster name
		}

		// discover service accounts and pod identity associations for each cluster
		for _, clusterName := range clusterNames {
			saList, err := a.client.listServiceAccounts(c.Self, c.RoleArn, c.Region,
				clusterName, c.Annotation)
			if err != nil {
				errorf("failed to list service accounts for cluster %s: %v",
					clusterName, err)
				continue // skip this cluster
			}

			piaList, err := a.client.listPodIdentityAssociations(c.Self, c.RoleArn,
				c.Region, clusterName)
			if err != nil {
				errorf("failed to list pod identity associations for cluster %s: %v",
					clusterName, err)
				continue // skip this cluster
			}

			saList = serviceAccountsExcludeNamespace(saList, c.ExcludeNamespaces)
			piaList = podIdentityAssociationExcludeNamespace(piaList, c.ExcludeNamespaces)

			cc := c
			cc.ClusterName = clusterName // discovered cluster name

			clusterList = append(clusterList, cluster{
				Config:                  cc,
				ServiceAccounts:         saList,
				PodIdentityAssociations: piaList,
			})
		}
	}

	return clusterList
}

func serviceAccountsExcludeNamespace(list []serviceAccount, exclude []string) []serviceAccount {
	var result []serviceAccount

LOOP:
	for _, sa := range list {
		for _, ns := range exclude {
			if sa.Namespace == ns {
				continue LOOP // exclude this SA
			}
		}
		result = append(result, sa) // keep this SA
	}

	return result
}

func podIdentityAssociationExcludeNamespace(list []podIdentityAssociation, exclude []string) []podIdentityAssociation {
	var result []podIdentityAssociation

LOOP:
	for _, pia := range list {
		for _, ns := range exclude {
			if pia.ServiceAccountNamespace == ns {
				continue LOOP // exclude this PIA
			}
		}
		result = append(result, pia) // keep this PIA
	}

	return result
}

type clientInterface interface {
	listEKSClusters(roleArn, region string) ([]string, error)

	// listServiceAccounts with self=true uses local client configuration.
	// with self=false, it builds client configuration for EKS using roleArn.
	// note that self=true requires exact clusterName provided in config as
	// cluster_name to be used in the other methods, since the clusterName
	// cannot be discovered.
	listServiceAccounts(self bool, roleArn, region,
		clusterName, annotationKey string) ([]serviceAccount, error)

	listPodIdentityAssociations(self bool, roleArn, region,
		clusterName string) ([]podIdentityAssociation, error)

	createPodIdentityAssociation(self bool, roleArn, region,
		clusterName, serviceAccountName, serviceAccountRoleArn string) error

	deletePodIdentityAssociation(self bool, roleArn, region,
		clusterName, associationID string) error
}

type serviceAccount struct {
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace"`
	AwsRoleArn string `yaml:"aws_role_arn"`
}

type podIdentityAssociation struct {
	AssociationID           string `yaml:"association_id"`
	ClusterName             string `yaml:"cluster_name"`
	ServiceAccountNamespace string `yaml:"service_account_namespace"`
	ServiceAccountName      string `yaml:"service_account_name"`
	RoleArn                 string `yaml:"role_arn"`
}
