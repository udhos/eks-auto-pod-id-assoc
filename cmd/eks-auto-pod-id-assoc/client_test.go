package main

import (
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"k8s.io/client-go/kubernetes"
)

// go test -race -run '^TestEksClientConcurrencyRace$' ./...
func TestEksClientConcurrencyRace(_ *testing.T) {

	const dry = false
	const namespace = ""
	const sampleRate = 1.0
	const dogstatsEnable = false

	met := newMetrics(namespace, defaultLatencyBucketsSeconds, sampleRate,
		dogstatsEnable)

	c := newRealClient("test", dry, met, fakeEksClient, fakeKubeClient)

	const concurrency = 10000

	var wg sync.WaitGroup

	const roleArn = "role1"
	const region = "region1"

	for range concurrency {
		wg.Go(func() {
			c.getEKSClient(roleArn, region)
		})
	}

	wg.Wait()
}

func fakeEksClient(_, _, _ string) (*eks.Client, error) {
	return &eks.Client{}, nil
}

// go test -race -run '^TestKubeClientConcurrencyRace$' ./...
func TestKubeClientConcurrencyRace(_ *testing.T) {

	const dry = false
	const namespace = ""
	const sampleRate = 1.0
	const dogstatsEnable = false

	met := newMetrics(namespace, defaultLatencyBucketsSeconds, sampleRate,
		dogstatsEnable)

	c := newRealClient("test", dry, met, fakeEksClient, fakeKubeClient)

	const concurrency = 10000

	var wg sync.WaitGroup

	const self = true
	const roleArn = "role1"
	const region = "region1"
	const clusterName = "cluster1"

	for range concurrency {
		wg.Go(func() {
			c.getKubeClient(self, roleArn, region, clusterName)
		})
	}

	wg.Wait()
}

func fakeKubeClient() (*kubernetes.Clientset, error) {
	return &kubernetes.Clientset{}, nil
}
