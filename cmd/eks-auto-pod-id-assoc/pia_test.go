package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func slogLevelError() {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})
	slog.SetDefault(slog.New(handler))
}

// go test -v -count 1 -run '^TestCreatePodIdentityAssociations$' ./...
func TestCreatePodIdentityAssociations(t *testing.T) {

	slogLevelError()

	const delay = 10 * time.Millisecond

	client := &clientPIAMock{delay: delay}

	var list []serviceAccount

	const serviceAccounts = 100

	for range serviceAccounts {
		list = append(list, serviceAccount{})
	}

	const (
		sampleRate      = 1.0
		dogstatsdEnable = false
	)

	m := newMetrics("ns", defaultLatencyBucketsSeconds, sampleRate, dogstatsdEnable)

	const (
		self          = false
		roleArn       = "role1"
		region        = "region1"
		cluster       = "cluster1"
		maxGoroutines = 1
	)

	begin := time.Now()

	createPodIdentityAssociations(context.TODO(), client, list, m, self,
		roleArn, region, cluster, defaultTags, maxGoroutines)

	elapsed := time.Since(begin)

	begin2 := time.Now()

	const manyGoroutines = defaultMaxConcurrency

	createPodIdentityAssociations(context.TODO(), client, list, m, self,
		roleArn, region, cluster, defaultTags, manyGoroutines)

	elapsed2 := time.Since(begin2)

	if elapsed < elapsed2 {
		t.Errorf("%d service accounts x %v delay: 1 goroutine elapsed: %v < %d goroutines elapsed: %v",
			serviceAccounts, delay, elapsed, manyGoroutines, elapsed2)
	}

	t.Logf("%d service accounts x %v delay: 1 goroutine elapsed: %v, %d goroutines elapsed: %v",
		serviceAccounts, delay, elapsed, manyGoroutines, elapsed2)
}

// go test -v -count 1 -run '^TestDeletePodIdentityAssociations$' ./...
func TestDeletePodIdentityAssociations(t *testing.T) {

	slogLevelError()

	const delay = 10 * time.Millisecond

	client := &clientPIAMock{delay: delay}

	var list []podIdentityAssociation

	const associations = 100

	for range associations {
		list = append(list, podIdentityAssociation{})
	}

	const (
		sampleRate      = 1.0
		dogstatsdEnable = false
	)

	m := newMetrics("ns", defaultLatencyBucketsSeconds, sampleRate, dogstatsdEnable)

	const (
		self          = false
		roleArn       = "role1"
		region        = "region1"
		cluster       = "cluster1"
		maxGoroutines = 1
	)

	begin := time.Now()

	deletePodIdentityAssociations(context.TODO(), client, list, m, self,
		roleArn, region, cluster, maxGoroutines)

	elapsed := time.Since(begin)

	begin2 := time.Now()

	const manyGoroutines = defaultMaxConcurrency

	deletePodIdentityAssociations(context.TODO(), client, list, m, self,
		roleArn, region, cluster, manyGoroutines)

	elapsed2 := time.Since(begin2)

	if elapsed < elapsed2 {
		t.Errorf("%d associations x %v delay: 1 goroutine elapsed: %v < %d goroutines elapsed: %v",
			associations, delay, elapsed, manyGoroutines, elapsed2)
	}

	t.Logf("%d associations x %v delay: 1 goroutine elapsed: %v, %d goroutines elapsed: %v",
		associations, delay, elapsed, manyGoroutines, elapsed2)
}

type clientPIAMock struct {
	delay time.Duration
}

func (m *clientPIAMock) createPodIdentityAssociation(self bool, roleArn, region,
	clusterName, serviceAccountNamespace, serviceAccountName,
	serviceAccountRoleArn string, tags map[string]string) error {
	time.Sleep(m.delay)
	return nil
}

func (m *clientPIAMock) deletePodIdentityAssociation(self bool, roleArn, region,
	clusterName, associationID string) error {
	time.Sleep(m.delay)
	return nil
}
