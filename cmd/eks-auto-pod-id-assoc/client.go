package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/udhos/boilerplate/awsconfig"
	"github.com/udhos/eks/eksclient"
	"github.com/udhos/kube/kubeclient"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type clientPIA interface {
	createPodIdentityAssociation(self bool, roleArn, region,
		clusterName, serviceAccountNamespace, serviceAccountName,
		serviceAccountRoleArn string, tags map[string]string) error

	deletePodIdentityAssociation(self bool, roleArn, region,
		clusterName, associationID string) error
}

type clientInterface interface {
	getKubeClient(self bool, roleArn,
		region, clusterName string) (*kubernetes.Clientset, error)

	listEKSClusters(roleArn, region string) ([]string, error)

	// listServiceAccounts with self=true uses local client configuration.
	// with self=false, it builds client configuration for EKS using roleArn.
	// note that self=true requires exact clusterName provided in config as
	// cluster_name to be used in the other methods, since the clusterName
	// cannot be discovered.
	listServiceAccounts(self bool, roleArn, region,
		clusterName, annotationKey string) ([]serviceAccount, error)

	listPodIdentityAssociations(self bool, roleArn, region,
		clusterName string, tags map[string]string,
		purgeExternalStaleAssociations bool) ([]podIdentityAssociation, error)

	clientPIA
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
}

func newRealClient(prog string, dry bool, metrics metrics) *realClient {
	return &realClient{
		prog:            prog,
		dry:             dry,
		metrics:         metrics,
		kubeClientCache: map[string]*kubernetes.Clientset{},
		eksClientCache:  map[string]*eks.Client{},
	}
}

type realClient struct {
	prog            string
	dry             bool
	metrics         metrics
	kubeClientCache map[string]*kubernetes.Clientset
	eksClientCache  map[string]*eks.Client
}

func kubeClientCacheKey(self bool, roleArn, region,
	clusterName string) string {
	return fmt.Sprintf("%t-%s-%s-%s", self, roleArn, region, clusterName)
}

func (c *realClient) getKubeClientCache(self bool, roleArn, region,
	clusterName string) *kubernetes.Clientset {
	key := kubeClientCacheKey(self, roleArn, region, clusterName)
	return c.kubeClientCache[key]
}

func (c *realClient) putKubeClientCache(self bool, roleArn, region,
	clusterName string, clientset *kubernetes.Clientset) {
	key := kubeClientCacheKey(self, roleArn, region, clusterName)
	c.kubeClientCache[key] = clientset
}

func (c *realClient) getEKSClient(roleArn, region string) (*eks.Client, error) {

	cacheKey := fmt.Sprintf("%s-%s", roleArn, region)
	if eksClient := c.eksClientCache[cacheKey]; eksClient != nil {
		return eksClient, nil
	}

	options := awsconfig.Options{
		Region:          region,
		RoleArn:         roleArn,
		RoleSessionName: c.prog,
	}
	awsCfg, errCfg := awsconfig.AwsConfig(options)
	if errCfg != nil {
		return nil, fmt.Errorf("could not get aws config: %w", errCfg)
	}

	eksClient := eks.NewFromConfig(awsCfg.AwsConfig)

	c.eksClientCache[cacheKey] = eksClient

	return eksClient, nil
}

func (c *realClient) listEKSClusters(roleArn,
	region string) ([]string, error) {

	const me = "listEKSClusters"

	clientEks, errEks := c.getEKSClient(roleArn, region)
	if errEks != nil {
		return nil, fmt.Errorf("%s: could not get EKS client: %w", me, errEks)
	}

	var maxResults int32 = 100 // 1..100

	paginator := eks.NewListClustersPaginator(clientEks, &eks.ListClustersInput{
		MaxResults: aws.Int32(maxResults),
	})

	var clusterNames []string

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%s: failed to get page: %w", me, err)
		}
		clusterNames = append(clusterNames, page.Clusters...)
	}

	infof("%s: region=%q found clusters: %d", me, region, len(clusterNames))

	return clusterNames, nil
}

func (c *realClient) getKubeClient(self bool, roleArn,
	region, clusterName string) (*kubernetes.Clientset, error) {

	if clientset := c.getKubeClientCache(self, roleArn, region,
		clusterName); clientset != nil {
		return clientset, nil
	}

	if self {
		// we are running in-cluster or with .kube/config
		clientset, err := kubeclient.New(kubeclient.Options{})
		if err == nil {
			c.putKubeClientCache(self, roleArn, region, clusterName,
				clientset)
		}
		return clientset, err
	}

	// do not attempt in-cluster or .kube/config,
	// generate kube client directly from eks

	clientEks, errEks := c.getEKSClient(roleArn, region)
	if errEks != nil {
		return nil, fmt.Errorf("could not get EKS client: %w", errEks)
	}

	//
	// get cluster data from eks client: CA data, endpoint
	//

	input := eks.DescribeClusterInput{Name: aws.String(clusterName)}

	begin := time.Now()

	out, errDesc := clientEks.DescribeCluster(context.TODO(), &input)

	elap := time.Since(begin)

	c.metrics.recordAPILatency(clusterName,
		apiEksDescribeCluster, getAPIStatus(errDesc),
		elap)

	infof("DescribeCluster: %s: elapsed=%v", clusterName, elap)

	if errDesc != nil {
		return nil, fmt.Errorf("describe eks cluster error: %w", errDesc)
	}

	clusterCAData := aws.ToString(out.Cluster.CertificateAuthority.Data)
	clusterEndpoint := aws.ToString(out.Cluster.Endpoint)

	//
	// create k8s client (clientset) from cluster data
	//

	eksclientOptions := eksclient.Options{
		ClusterName:     clusterName,
		ClusterCAData:   clusterCAData,
		ClusterEndpoint: clusterEndpoint,
	}

	clientset, err := eksclient.New(eksclientOptions)
	if err == nil {
		c.putKubeClientCache(self, roleArn, region, clusterName, clientset)
	}
	return clientset, err
}

func (c *realClient) listServiceAccounts(self bool, roleArn, region,
	clusterName, annotationKey string) ([]serviceAccount, error) {

	const me = "listServiceAccounts"

	clientset, err := c.getKubeClient(self, roleArn, region, clusterName)
	if err != nil {
		return nil, fmt.Errorf("%s: kube client error: %w", me, err)
	}

	const namespace = ""

	list, errList := clientset.CoreV1().ServiceAccounts(namespace).List(context.TODO(), v1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("%s: list service accounts error: %w", me, errList)
	}

	var result []serviceAccount

	if annotationKey == "" {
		annotationKey = "eks.amazonaws.com/role-arn"
	}

	for _, item := range list.Items {

		annotations := item.GetObjectMeta().GetAnnotations()
		if annotations == nil {
			continue
		}
		value, found := annotations[annotationKey]
		if !found {
			continue
		}

		sa := serviceAccount{
			Namespace:  item.Namespace,
			Name:       item.Name,
			AwsRoleArn: value,
		}
		result = append(result, sa)
	}

	infof("%s: region=%q cluster=%q annotation=%s found service accounts: annotated=%d total=%d",
		me, region, clusterName, annotationKey, len(result), len(list.Items))

	return result, nil
}

func (c *realClient) listPodIdentityAssociations(_ bool, roleArn, region,
	clusterName string, tags map[string]string,
	purgeExternalStaleAssociations bool) ([]podIdentityAssociation, error) {

	const me = "listPodIdentityAssociations"

	clientEks, errEks := c.getEKSClient(roleArn, region)
	if errEks != nil {
		return nil, fmt.Errorf("%s: could not get EKS client: %w", me, errEks)
	}

	var maxResults int32 = 100 // 1..100

	paginator := eks.NewListPodIdentityAssociationsPaginator(clientEks,
		&eks.ListPodIdentityAssociationsInput{
			ClusterName: aws.String(clusterName),
			MaxResults:  aws.Int32(maxResults),
		})

	var piaList []podIdentityAssociation

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("%s: failed to get page: %w", me, err)
		}
		for _, assoc := range page.Associations {
			pia := podIdentityAssociation{
				AssociationID:           aws.ToString(assoc.AssociationId),
				ClusterName:             aws.ToString(assoc.ClusterName),
				ServiceAccountNamespace: aws.ToString(assoc.Namespace),
				ServiceAccountName:      aws.ToString(assoc.ServiceAccount),
			}
			piaList = append(piaList, pia)
		}
	}

	infof("%s: region=%q cluster=%q found pod identity associations: %d",
		me, region, clusterName, len(piaList))

	return piaList, nil
}

var defaultTags = map[string]string{
	"managed-by": "eks-auto-pod-id-assoc",
}

func (c *realClient) createPodIdentityAssociation(_ bool, roleArn, region,
	clusterName, serviceAccountNamespace, serviceAccountName,
	serviceAccountRoleArn string, tags map[string]string) error {

	const me = "createPodIdentityAssociation"

	clientEks, errEks := c.getEKSClient(roleArn, region)
	if errEks != nil {
		return fmt.Errorf("%s: could not get EKS client: %w", me, errEks)
	}

	input := &eks.CreatePodIdentityAssociationInput{
		ClusterName:    aws.String(clusterName),
		Namespace:      aws.String(serviceAccountNamespace),
		RoleArn:        aws.String(serviceAccountRoleArn),
		ServiceAccount: aws.String(serviceAccountName),
		Tags:           tags,
	}

	var resp *eks.CreatePodIdentityAssociationOutput
	var err error

	if !c.dry {
		resp, err = clientEks.CreatePodIdentityAssociation(context.TODO(), input)
	}

	if err != nil {
		return fmt.Errorf("%s: error: %w", me, err)
	}

	var associationID string
	if c.dry {
		associationID = "<dry>"
	} else {
		associationID = aws.ToString(resp.Association.AssociationId)
	}

	infof("%s: dry=%t region=%q created pod identity association: associationId=%s cluster=%s serviceAccount=%s namespace=%s role=%s",
		me, c.dry, region,
		associationID,
		clusterName,
		serviceAccountName,
		serviceAccountNamespace,
		serviceAccountRoleArn)

	return nil
}

func (c *realClient) deletePodIdentityAssociation(_ bool, roleArn, region,
	clusterName, associationID string) error {

	const me = "deletePodIdentityAssociation"

	clientEks, errEks := c.getEKSClient(roleArn, region)
	if errEks != nil {
		return fmt.Errorf("%s: could not get EKS client: %w", me, errEks)
	}

	input := &eks.DeletePodIdentityAssociationInput{
		ClusterName:   aws.String(clusterName),
		AssociationId: aws.String(associationID),
	}

	var err error

	if !c.dry {
		_, err = clientEks.DeletePodIdentityAssociation(context.TODO(), input)
	}

	if err != nil {
		return fmt.Errorf("%s: error: %w", me, err)
	}

	infof("%s: dry=%t region=%q deleted pod identity association: associationId=%s cluster=%s",
		me, c.dry, region,
		associationID,
		clusterName)

	return nil
}
