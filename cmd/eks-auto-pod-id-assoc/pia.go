package main

import (
	"context"
	"fmt"
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

	if maxGoroutines < 1 {
		maxGoroutines = defaultMaxConcurrency
	}

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
			m.recordAPILatency(clusterName,
				apiEksCreatePodIdentityAssociation, getAPIStatus(err),
				elap)

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

	if maxGoroutines < 1 {
		maxGoroutines = defaultMaxConcurrency
	}

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
			m.recordAPILatency(clusterName,
				apiEksDeletePodIdentityAssociation, getAPIStatus(err),
				elap)

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
