package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/udhos/dogstatsdclient/dogstatsdclient"
)

type metrics struct {
	statsdClient *dogstatsdclient.Client
	sampleRate   float64
	registry     *prometheus.Registry

	serviceAccounts         *prometheus.GaugeVec
	podIdentityAssociations *prometheus.GaugeVec
	discoverLatency         *prometheus.GaugeVec
	reconcileLatency        *prometheus.GaugeVec
	apiLatency              *prometheus.HistogramVec
}

const (
	metricServiceAccounts         = "service_accounts"
	metricPodIdentityAssociations = "pod_identity_associations"
	metricDiscoverLatency         = "discover_latency_seconds"
	metricReconcileLatency        = "reconcile_latency_seconds"
	metricAPILatency              = "api_latency_seconds"

	labelKeyCluster      = "cluster"
	labelKeyIgnoreReason = "ignore_reason"
	labelKeyAPI          = "api"
	labelKeyStatus       = "status"
)

func tag(key, value string) string {
	return key + ":" + value
}

func (m metrics) recordServiceAccounts(cluster,
	ignoreReason string, value float64) {

	if m.statsdClient != nil {
		tags := []string{tag(labelKeyCluster, cluster),
			tag(labelKeyIgnoreReason, ignoreReason)}
		m.statsdClient.Gauge(metricServiceAccounts, value, tags, m.sampleRate)
	}

	m.serviceAccounts.WithLabelValues(cluster,
		ignoreReason).Set(value)
}

func (m metrics) recordPodIdentityAssociations(cluster,
	ignoreReason string, value float64) {

	if m.statsdClient != nil {
		tags := []string{tag(labelKeyCluster, cluster),
			tag(labelKeyIgnoreReason, ignoreReason)}
		m.statsdClient.Gauge(metricPodIdentityAssociations, value, tags,
			m.sampleRate)
	}

	m.podIdentityAssociations.WithLabelValues(cluster,
		ignoreReason).Set(value)
}

func (m metrics) recordDiscoverLatency(cluster string,
	elapsed time.Duration) {
	sec := float64(elapsed) / float64(time.Second)

	if m.statsdClient != nil {
		tags := []string{tag(labelKeyCluster, cluster)}
		m.statsdClient.Distribution(metricDiscoverLatency, sec, tags,
			m.sampleRate)
	}

	m.discoverLatency.WithLabelValues(cluster).Set(sec)
}

func (m metrics) recordReconcileLatency(cluster string,
	elapsed time.Duration) {
	sec := float64(elapsed) / float64(time.Second)

	if m.statsdClient != nil {
		tags := []string{tag(labelKeyCluster, cluster)}
		m.statsdClient.Distribution(metricReconcileLatency, sec, tags,
			m.sampleRate)
	}

	m.reconcileLatency.WithLabelValues(cluster).Set(sec)
}

func (m metrics) recordAPILatency(cluster, api, status string,
	elapsed time.Duration) {
	sec := float64(elapsed) / float64(time.Second)

	if m.statsdClient != nil {
		tags := []string{tag(labelKeyCluster, cluster), tag(labelKeyAPI, api),
			tag(labelKeyStatus, status)}
		m.statsdClient.Distribution(metricAPILatency, sec, tags,
			m.sampleRate)
	}

	m.apiLatency.WithLabelValues(cluster, api, status).Observe(sec)
}

const (
	ignoreReasonNotIgnored     = "not_ignored"
	ignoreReasonExcluded       = "excluded"
	ignoreReasonRestrictedRole = "restricted_role"

	apiServiceAccountsList             = "serviceaccounts.list"
	apiEksListClusters                 = "eks:ListClusters"
	apiEksDescribeCluster              = "eks:DescribeCluster"
	apiEksListPodIdentityAssociations  = "eks:ListPodIdentityAssociations"
	apiEksCreatePodIdentityAssociation = "eks:CreatePodIdentityAssociation"
	apiEksDeletePodIdentityAssociation = "eks:DeletePodIdentityAssociation"

	apiStatusOk    = "ok"
	apiStatusError = "error"
)

func getAPIStatus(err error) string {
	if err == nil {
		return apiStatusOk
	}
	return apiStatusError
}

func newMetrics(namespace string, latencyBucketsSeconds []float64,
	sampleRate float64, dogstatsdEnable bool) metrics {

	registry := prometheus.NewRegistry()

	const subsystem = ""

	var c *dogstatsdclient.Client

	if dogstatsdEnable {
		var errClient error
		c, errClient = dogstatsdclient.New(dogstatsdclient.Options{
			Namespace: namespace,
		})
		if errClient != nil {
			errorf("newMetrics: dogstatsd client error: %v", errClient)
		}
	}

	return metrics{
		statsdClient: c,
		sampleRate:   sampleRate,

		registry: registry,

		serviceAccounts: newGaugeVec(
			registry,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      metricServiceAccounts,
				Help:      "Number of Service Accounts.",
			},
			[]string{labelKeyCluster, labelKeyIgnoreReason},
		),

		podIdentityAssociations: newGaugeVec(
			registry,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      metricPodIdentityAssociations,
				Help:      "Number of Pod Identity Associations.",
			},
			[]string{labelKeyCluster, labelKeyIgnoreReason},
		),

		discoverLatency: newGaugeVec(
			registry,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      metricDiscoverLatency,
				Help:      "Latency of discovery.",
			},
			[]string{labelKeyCluster},
		),

		reconcileLatency: newGaugeVec(
			registry,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      metricReconcileLatency,
				Help:      "Latency of reconcile.",
			},
			[]string{labelKeyCluster},
		),

		apiLatency: newHistogramVec(
			registry,
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      metricAPILatency,
				Help:      "Latency of API calls in seconds.",
				Buckets:   latencyBucketsSeconds,
			},
			[]string{labelKeyCluster, labelKeyAPI, labelKeyStatus},
		),
	}
}

func newGaugeVec(registerer prometheus.Registerer,
	opts prometheus.GaugeOpts,
	labelValues []string) *prometheus.GaugeVec {
	return promauto.With(registerer).NewGaugeVec(opts, labelValues)
}

func newHistogramVec(registerer prometheus.Registerer,
	opts prometheus.HistogramOpts,
	labelValues []string) *prometheus.HistogramVec {
	return promauto.With(registerer).NewHistogramVec(opts, labelValues)
}

func (a *application) serveMetrics(path string) {

	registry := a.metrics.registry

	handler := promhttp.InstrumentMetricHandler(
		registry, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	a.server.mux.Handle(path, handler)
}
