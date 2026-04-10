package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/udhos/kube-informer-serviceaccount/serviceaccountinformer"
	"go.yaml.in/yaml/v4"
)

type application struct {
	cfg        config
	client     clientInterface
	server     httpServer
	metrics    metrics
	informer   map[string]*informer
	informerCh chan struct{}
}

type informer struct {
	informer *serviceaccountinformer.ServiceAccountInformer
	stale    bool
}

func newApplication(cfg config, met metrics,
	client clientInterface) *application {
	app := &application{
		cfg:        cfg,
		client:     client,
		metrics:    met,
		informer:   map[string]*informer{},
		informerCh: make(chan struct{}),
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

	a.updateInformers(clusterList)

	a.reconcileClusters(clusterList)
}

func (a *application) updateInformers(clusterList []cluster) {

	// mark all informers as stale
	for _, inf := range a.informer {
		inf.stale = true
	}

	// create new informers
	for _, cl := range clusterList {
		key := fmt.Sprintf("%s %s %s", cl.Config.RoleArn, cl.Config.Region, cl.Config.ClusterName)
		inf, found := a.informer[key]

		if found {
			inf.stale = false // found, not stale
			continue
		}

		// create new informer

		clientset, errKubeclient := a.client.getKubeClient(cl.Config.Self,
			cl.Config.RoleArn, cl.Config.Region, cl.Config.ClusterName)
		if errKubeclient != nil {
			errorf("updateInformers: could not get kube client: self=%t role=%q region=%q cluster=%q: %v",
				cl.Config.Self, cl.Config.RoleArn, cl.Config.Region, cl.Config.ClusterName, errKubeclient)
			continue
		}

		options := serviceaccountinformer.Options{
			Client: clientset,
			OnUpdate: func(serviceAccounts []serviceaccountinformer.ServiceAccount) {
				debugf("OnUpdate: service accounts: %d", len(serviceAccounts))

				// trigger cycle
				select {
				case a.informerCh <- struct{}{}:
				// Signal sent successfully
				default:
					// Signal already pending in the unbuffered channel;
					// because we use a boolean flag on the receiving side,
					// we don't need to queue more.
				}
			},
		}

		newInf := serviceaccountinformer.New(options)

		go func() {
			errRun := newInf.Run()
			infof("updateInformers: informer exited: self=%t role=%q region=%q cluster=%q: error:%v",
				cl.Config.Self, cl.Config.RoleArn, cl.Config.Region, cl.Config.ClusterName, errRun)
		}()

		a.informer[key] = &informer{informer: newInf}

		infof("updateInformers: informer started: self=%t role=%q region=%q cluster=%q",
			cl.Config.Self, cl.Config.RoleArn, cl.Config.Region, cl.Config.ClusterName)
	}

	// remove stale informers
	for k, inf := range a.informer {
		if inf.stale {
			inf.informer.Stop()
			a.informer[k] = nil // help GC
			delete(a.informer, k)
			infof("updateInformers: stale informer deleted: %s", k)
		}
	}
}

func dumpClusters(clusterList []cluster, label string) {

	if !logEmitDebug() {
		return
	}

	// slog by default writes to stderr, so we use it here.
	// however if slog output is changed, we would be out of sync.
	out := os.Stderr

	fmt.Fprintln(out, label)
	yamlBytes, _ := yaml.Marshal(clusterList)
	fmt.Fprintln(out, string(yamlBytes))
}

func (a *application) reconcileOneCluster(cl cluster) {

	clusterLabel := getClusterLabel(cl.Config.RoleArn, cl.Config.Region,
		cl.Config.ClusterName)

	var wg sync.WaitGroup

	wg.Go(func() {
		// create associations for service accounts without associations

		missingServiceAccounts := a.findMissingServiceAccounts(cl)

		infof("%s found missing service accounts: %d",
			clusterLabel, len(missingServiceAccounts))

		createPodIdentityAssociations(context.TODO(),
			a.client, missingServiceAccounts,
			a.metrics, cl.Config.Self, cl.Config.RoleArn,
			cl.Config.Region, cl.Config.ClusterName,
			cl.Config.PodIdentityAssociationTags,
			cl.Config.MaxConcurrency)
	})

	// delete associations for service accounts that don't exist

	stalePIAs := a.findStalePodIdentityAssociations(cl)

	infof("%s found stale pod identity associations: %d",
		clusterLabel, len(stalePIAs))

	deletePodIdentityAssociations(context.TODO(),
		a.client, stalePIAs, a.metrics,
		cl.Config.Self, cl.Config.RoleArn, cl.Config.Region,
		cl.Config.ClusterName, cl.Config.MaxConcurrency)

	wg.Wait() // wait creations
}

func (a *application) reconcileClusters(clusterList []cluster) {
	for _, cl := range clusterList {

		begin := time.Now()

		a.reconcileOneCluster(cl)

		elap := time.Since(begin)

		a.metrics.recordReconcileLatency(cl.Config.ClusterName, elap)

		infof("reconcile latency: region=%q cluster=%q elapsed=%v",
			cl.Config.Region, cl.Config.ClusterName, elap)
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
	pattern, errPattern := newPattern(c.ClusterName)
	if errPattern != nil {
		return nil, fmt.Errorf("failed to compile cluster name pattern %s: %w",
			c.ClusterName, errPattern)
	}

	// list clusters in the region

	begin := time.Now()

	names, err := a.client.listEKSClusters(c.RoleArn, c.Region)

	elap := time.Since(begin)

	a.metrics.recordAPILatency("",
		apiEksListClusters, getAPIStatus(err),
		elap)

	infof("ListClusters region=%q elapsed=%v found=%d: error: %v",
		c.Region, elap, len(names), err)

	if err != nil {
		return nil, fmt.Errorf("failed to list EKS clusters in region %s: %w",
			c.Region, err)
	}

	// pick only matching patterns
	for _, name := range names {
		match := pattern.match(name)
		infof("region=%s pattern=%q cluster_name=%s matched=%t",
			c.Region, c.ClusterName, name, match)
		if !match {
			continue // skip this cluster name
		}
		clusterNames = append(clusterNames, name)
	}

	return clusterNames, nil
}

func (a *application) listAssociations(self bool, roleArn, region,
	clusterName string, tags map[string]string,
	purgeExternalStaleAssociations,
	forceIterativeAssociationDiscovery bool,
	maxConcurrency int) ([]podIdentityAssociation, error) {

	const me = "application.listAssociations"

	// get full associations list with listAssociations
	allAssocs, errList := a.client.listPodIdentityAssociations(self, roleArn,
		region, clusterName, a.metrics)
	if errList != nil {
		return nil, errList
	}

	// If purge is enabled, we don't care about tags; return everything.
	if purgeExternalStaleAssociations {
		return allAssocs, nil
	}

	if forceIterativeAssociationDiscovery {
		return listTaggedPodIdentityAssociationsWithDescribe(context.TODO(), a.client,
			allAssocs, a.metrics, self, roleArn, region, clusterName, tags, maxConcurrency)
	}

	// query GetResources for all associations with our tags
	taggedAssocIDs, errTags := a.client.listTaggedAssociationIDs(roleArn, clusterName,
		region, tags, a.metrics)
	if errTags != nil {
		return nil, fmt.Errorf("%s: failed to get tagged resources: %w", me, errTags)
	}

	debugf("%s: found tagged associations: %d", me, len(taggedAssocIDs))

	for _, id := range taggedAssocIDs {
		debugf("%s: found tagged association: %s", me, id)
	}

	// pick only associations that appeared in our tagged list
	var result []podIdentityAssociation
	for _, assoc := range allAssocs {
		for _, tagged := range taggedAssocIDs {
			if assoc.AssociationID == tagged {
				result = append(result, assoc)
			}
		}
	}

	infof("%s: region=%q cluster=%q filtered tagged associations: %d/%d",
		me, region, clusterName, len(result), len(allAssocs))

	return result, nil
}

func (a *application) discoverOneCluster(c configCluster, clusterName string) (cluster, error) {
	beginSA := time.Now()

	//
	// service accounts
	//

	saList, errSA := a.client.listServiceAccounts(c.Self, c.RoleArn, c.Region,
		clusterName, c.Annotation)

	elapsedSA := time.Since(beginSA)

	a.metrics.recordAPILatency(clusterName,
		apiServiceAccountsList, getAPIStatus(errSA),
		elapsedSA)

	if errSA != nil {
		return cluster{}, fmt.Errorf("failed to list service accounts for cluster %s: elapsed=%v: %w",
			clusterName, elapsedSA, errSA)
	}

	saTotal := len(saList)

	//
	// pod identity associations
	//

	infof("listServiceAccounts: cluster=%q elapsed=%v found=%d",
		clusterName, elapsedSA, saTotal)

	beginPIA := time.Now()

	piaList, errPIA := a.listAssociations(c.Self, c.RoleArn, c.Region,
		clusterName, c.PodIdentityAssociationTags,
		c.PurgeExternalStaleAssociations,
		c.ForceIterativeAssociationDiscovery, c.MaxConcurrency)

	elapsedPIA := time.Since(beginPIA)

	if errPIA != nil {
		return cluster{}, fmt.Errorf("failed to list pod identity associations for cluster %s: elapsed=%v: %w",
			clusterName, elapsedPIA, errPIA)
	}

	piaTotal := len(piaList)

	infof("listPodIdentityAssociations: cluster=%q elapsed=%v found=%d",
		clusterName, elapsedPIA, piaTotal)

	// exclude SAs according RestrictRoles
	saList = excludeRestrictedRoles(saList, c.RestrictRoles)

	saNonRestricted := len(saList)
	saRestricted := saTotal - saNonRestricted

	// exclude SAs according exclude_service_accounts
	saList = serviceAccountsExcludeServiceAccounts(saList, c.ExcludeServiceAccounts)

	saNonIgnored := len(saList)
	saExcluded := saNonRestricted - saNonIgnored

	a.metrics.recordServiceAccounts(clusterName, ignoreReasonNotIgnored, float64(saNonIgnored))
	a.metrics.recordServiceAccounts(clusterName, ignoreReasonExcluded, float64(saExcluded))
	a.metrics.recordServiceAccounts(clusterName, ignoreReasonRestrictedRole, float64(saRestricted))

	// exclude PIAs according exclude_service_accounts
	piaList = podIdentityAssociationExcludeServiceAccounts(piaList, c.ExcludeServiceAccounts)

	piaNonIgnored := len(piaList)
	piaExcluded := piaTotal - piaNonIgnored

	a.metrics.recordPodIdentityAssociations(clusterName, ignoreReasonNotIgnored, float64(piaNonIgnored))
	a.metrics.recordPodIdentityAssociations(clusterName, ignoreReasonExcluded, float64(piaExcluded))

	c.ClusterName = clusterName // discovered cluster name

	return cluster{
		Config:                  c,
		ServiceAccounts:         saList,
		PodIdentityAssociations: piaList,
	}, nil
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

			begin := time.Now()

			cl, errDisc := a.discoverOneCluster(c, clusterName)

			elap := time.Since(begin)

			a.metrics.recordDiscoverLatency(clusterName, elap)

			infof("discover latency: region=%q cluster=%q elapsed=%v",
				cl.Config.Region, cl.Config.ClusterName, elap)

			if errDisc != nil {
				errorf("%v", errDisc)
				continue
			}
			clusterList = append(clusterList, cl)
		}
	}

	return clusterList
}

// findRestrictedRole finds the rule that restricts a role arn, if any.
func findRestrictedRole(restrict []restrictRole, roleArn string) (restrictRole, bool) {
	for _, rr := range restrict {
		if rr.matchRole.match(roleArn) {
			return rr, true
		}
	}
	return restrictRole{}, false
}

// restrictedRoleAllowsServiceAccount checks if a restrict_roles rule allows a service account.
func restrictedRoleAllowsServiceAccount(rr restrictRole, serviceAccountName,
	serviceAccountNamespace string) bool {
	for _, allow := range rr.Allow {
		if allow.match(serviceAccountName, serviceAccountNamespace) {
			return true
		}
	}
	return false
}

// excludeRestrictedRoles returns a new slice containing only
// the service accounts that pass the restrict_roles rules.
func excludeRestrictedRoles(list []serviceAccount,
	restrict []restrictRole) []serviceAccount {

	var result []serviceAccount

	for _, sa := range list {

		// is the SA trying to use a restricted role?

		rr, found := findRestrictedRole(restrict, sa.AwsRoleArn)
		if !found {
			// not a restricted role
			result = append(result, sa) // keep this SA
			continue
		}

		// restricted role

		if allowed := restrictedRoleAllowsServiceAccount(rr, sa.Name,
			sa.Namespace); allowed {
			result = append(result, sa) // keep this SA
		}
	}

	return result
}

// matchesExclude returns true if any of the provided exclusion patterns
// match the given service account name/namespace pair. the logic is kept
// separate so the two filter functions remain small and easy to read.
func matchesExclude(name, namespace string,
	exclude []matchServiceAccount) bool {
	for _, ex := range exclude {
		if ex.match(name, namespace) {
			return true
		}
	}
	return false
}

// serviceAccountsExcludeServiceAccounts returns a new slice containing only
// the service accounts that do *not* match any exclusion pattern.
func serviceAccountsExcludeServiceAccounts(list []serviceAccount,
	exclude []matchServiceAccount) []serviceAccount {
	var result []serviceAccount
	for _, sa := range list {
		if matchesExclude(sa.Name, sa.Namespace, exclude) {
			continue // exclude this SA
		}
		result = append(result, sa) // keep this SA
	}
	return result
}

// podIdentityAssociationExcludeServiceAccounts filters PIAs using the same
// exclusion rules as above.
func podIdentityAssociationExcludeServiceAccounts(list []podIdentityAssociation,
	exclude []matchServiceAccount) []podIdentityAssociation {
	var result []podIdentityAssociation
	for _, pia := range list {
		if matchesExclude(pia.ServiceAccountName, pia.ServiceAccountNamespace, exclude) {
			continue // exclude this PIA
		}
		result = append(result, pia) // keep this PIA
	}
	return result
}
