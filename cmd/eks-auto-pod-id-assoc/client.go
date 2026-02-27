package main

import (
	"context"
	"errors"
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
		return kubeclient.New(kubeclient.Options{})
	}

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

func (c *realClient) listPodIdentityAssociations(self bool, roleArn, region,
	clusterName string) ([]podIdentityAssociation, error) {
	return []podIdentityAssociation{}, errors.New("not implemented")
}

func (c *realClient) createPodIdentityAssociation(self bool, roleArn, region,
	clusterName, serviceAccountName, serviceAccountRoleArn string) error {
	return errors.New("not implemented")
}

func (c *realClient) deletePodIdentityAssociation(self bool, roleArn, region,
	clusterName, associationID string) error {
	return errors.New("not implemented")
}
