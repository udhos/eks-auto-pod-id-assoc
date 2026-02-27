package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/udhos/boilerplate/awsconfig"
	"github.com/udhos/eks/eksclient"
	"github.com/udhos/kube/kubeclient"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func newRealClient(prog string, dry bool) *realClient {
	return &realClient{
		prog: prog,
		dry:  dry}
}

type realClient struct {
	prog string
	dry  bool
}

func (c *realClient) getEKSClient(roleArn, region string) (*eks.Client, error) {
	options := awsconfig.Options{
		Region:          region,
		RoleArn:         roleArn,
		RoleSessionName: c.prog,
	}
	awsCfg, errCfg := awsconfig.AwsConfig(options)
	if errCfg != nil {
		return nil, fmt.Errorf("could not get aws config: %w", errCfg)
	}
	return eks.NewFromConfig(awsCfg.AwsConfig), nil
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
	if self {
		// we are running in-cluster or with .kube/config
		return kubeclient.New(kubeclient.Options{})
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

	out, errDesc := clientEks.DescribeCluster(context.TODO(), &input)
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

	return eksclient.New(eksclientOptions)
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

	infof("%s: region=%q annotation=%s found service accounts: annotated=%d total=%d",
		me, region, annotationKey, len(result), len(list.Items))

	return result, nil
}

func (c *realClient) listPodIdentityAssociations(_ bool, roleArn, region,
	clusterName string) ([]podIdentityAssociation, error) {

	const me = "listPodIdentityAssociations"

	clientEks, errEks := c.getEKSClient(roleArn, region)
	if errEks != nil {
		return nil, fmt.Errorf("%s: could not get EKS client: %w", me, errEks)
	}

	var maxResults int32 = 100 // 1..100

	paginator := eks.NewListPodIdentityAssociationsPaginator(clientEks, &eks.ListPodIdentityAssociationsInput{
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

	infof("%s: region=%q found pod identity associations: %d", me, region, len(piaList))

	return piaList, nil
}

func (c *realClient) createPodIdentityAssociation(_ bool, roleArn, region,
	clusterName, serviceAccountNamespace, serviceAccountName, serviceAccountRoleArn string) error {

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
