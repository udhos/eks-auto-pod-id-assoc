package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

func getClusterLabel(roleArn, region, clusterName string) string {
	return fmt.Sprintf("role=%q region=%q cluster=%q",
		roleArn, region, clusterName)
}

func createPodIdentityAssociations(ctx context.Context,
	client clientPIA,
	list []serviceAccount, m metrics,
	self bool, roleArn, region, clusterName string,
	tags map[string]string, maxGoroutines int) {

	clusterLabel := getClusterLabel(roleArn, region, clusterName)

	// ErrGroup with a limit is the modern 'Sempahore + WaitGroup'
	// It also handles context cancellation automatically.
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(maxGoroutines)

	for i, sa := range list {
		g.Go(func() error {
			label := fmt.Sprintf("%d/%d", i+1, len(list))
			begin := time.Now()

			// If client supports Context, pass it through for cancellation support
			err := client.createPodIdentityAssociation(self, roleArn,
				region, clusterName, sa.Namespace, sa.Name, sa.AwsRoleArn,
				tags)

			elap := time.Since(begin)
			m.recordAPILatency(clusterName, apiEksCreatePodIdentityAssociation,
				getAPIStatus(err), elap)

			if err != nil {
				errorf("%s failure creating pod identity association %s: serviceAccount=%q serviceAccountRoleArn=%q elapsed=%v: %v",
					clusterLabel, label, sa.Name, sa.AwsRoleArn, elap, err)
				return nil
			}

			infof("%s created pod identity association %s: serviceAccount=%q serviceAccountRoleArn=%q elapsed=%v",
				clusterLabel, label, sa.Name, sa.AwsRoleArn, elap)

			return nil
		})
	}

	_ = g.Wait()
}

func deletePodIdentityAssociations(ctx context.Context,
	client clientPIA,
	list []podIdentityAssociation, m metrics,
	self bool, roleArn, region, clusterName string,
	maxGoroutines int) {

	clusterLabel := getClusterLabel(roleArn, region, clusterName)

	// ErrGroup manages the pool and the limit natively
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(maxGoroutines)

	for i, pia := range list {
		g.Go(func() error {
			label := fmt.Sprintf("%d/%d", i+1, len(list))
			begin := time.Now()

			// If client supports Context, pass it through for cancellation support
			err := client.deletePodIdentityAssociation(self, roleArn,
				region, clusterName, pia.AssociationID)

			elap := time.Since(begin)
			m.recordAPILatency(clusterName, apiEksDeletePodIdentityAssociation,
				getAPIStatus(err), elap)

			if err != nil {
				errorf("%s failure deleting pod identity association %s: associationID=%q serviceAccount=%q elapsed=%v: %v",
					clusterLabel, label, pia.AssociationID, pia.ServiceAccountName, elap, err)
				return nil
			}

			infof("%s deleted pod identity association %s: associationID=%q serviceAccount=%q elapsed=%v",
				clusterLabel, label, pia.AssociationID, pia.ServiceAccountName, elap)

			return nil
		})
	}

	_ = g.Wait()
}

func listTaggedPodIdentityAssociationsWithDescribe(ctx context.Context,
	client clientPIA,
	fullAssociationList []podIdentityAssociation, m metrics,
	self bool, roleArn, region, clusterName string,
	tags map[string]string,
	maxGoroutines int) ([]podIdentityAssociation, error) {

	clusterLabel := getClusterLabel(roleArn, region, clusterName)

	var result []podIdentityAssociation
	var mu sync.Mutex

	// ErrGroup manages the pool and the limit natively
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(maxGoroutines)

	for i, pia := range fullAssociationList {

		g.Go(func() error {
			label := fmt.Sprintf("%d/%d", i+1, len(fullAssociationList))

			begin := time.Now()

			assocTags, err := client.getPodIdentityAssociationTags(self, roleArn,
				region, clusterName, pia.AssociationID)

			elap := time.Since(begin)
			m.recordAPILatency(clusterName, apiEksDescribePodIdentityAssociation,
				getAPIStatus(err), elap)

			if err != nil {
				return fmt.Errorf("error describing association: %s: associationID=%s: error: %w",
					clusterLabel, pia.AssociationID, err)
			}

			debugf("%s getPodIdentityAssociationTags %s: associationID=%q serviceAccount=%q elapsed=%v",
				clusterLabel, label, pia.AssociationID, pia.ServiceAccountName, elap)

			if hasTags(assocTags, tags) {
				mu.Lock()
				result = append(result, pia)
				mu.Unlock()
			}

			return nil
		})
	}

	if errResult := g.Wait(); errResult != nil {
		return nil, errResult
	}

	return result, nil
}

func hasTags(tags, required map[string]string) bool {
	if len(required) == 0 {
		return true
	}
	for k, v := range required {
		vv, found := tags[k]
		if !found {
			return false // required tag key not found
		}
		if vv != v {
			return false // requied tag value not found
		}
	}
	return true
}
