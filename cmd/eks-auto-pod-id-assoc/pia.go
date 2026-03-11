package main

import (
	"fmt"
	"time"
)

func getClusterLabel(roleArn, region, clusterName string) string {
	return fmt.Sprintf("role=%q region=%q cluster=%q",
		roleArn, region, clusterName)
}

func createPodIdentityAssociations(client clientPIA,
	list []serviceAccount, m metrics,
	self bool, roleArn, region, clusterName string,
	tags map[string]string) {

	clusterLabel := getClusterLabel(roleArn, region, clusterName)

	for i, sa := range list {
		label := fmt.Sprintf("%d/%d", i+1, len(list))

		begin := time.Now()

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
			continue
		}

		infof("%s created pod identity association %s: serviceAccount=%q serviceAccountRoleArn=%q elapsed=%v",
			clusterLabel, label, sa.Name, sa.AwsRoleArn, elap)
	}

}

func deletePodIdentityAssociations(client clientPIA,
	list []podIdentityAssociation, m metrics,
	self bool, roleArn, region, clusterName string) {

	clusterLabel := getClusterLabel(roleArn, region, clusterName)

	for i, pia := range list {
		label := fmt.Sprintf("%d/%d", i+1, len(list))

		begin := time.Now()

		err := client.deletePodIdentityAssociation(self, roleArn,
			region, clusterName, pia.AssociationID)

		elap := time.Since(begin)

		m.recordAPILatency(clusterName,
			apiEksDeletePodIdentityAssociation, getAPIStatus(err),
			elap)

		if err != nil {
			errorf("%s failure deleting pod identity association %s: associationID=%q serviceAccount=%q elapsed=%v: %v",
				clusterLabel, label, pia.AssociationID, pia.ServiceAccountName, elap, err)
			continue
		}

		infof("%s deleted pod identity association %s: associationID=%q serviceAccount=%q elapsed=%v",
			clusterLabel, label, pia.AssociationID, pia.ServiceAccountName, elap)
	}

}
