package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	registry *prometheus.Registry

	serviceAccounts         *prometheus.GaugeVec
	podIdentityAssociations *prometheus.GaugeVec
	discoverLatency         *prometheus.GaugeVec
	reconcileLatency        *prometheus.GaugeVec
	apiLatency              *prometheus.HistogramVec
}

func (m metrics) recordServiceAccounts(cluster,
	ignoreReason string, value float64) {
	m.serviceAccounts.WithLabelValues(cluster,
		ignoreReason).Set(value)
}

func (m metrics) recordPodIdentityAssociations(cluster,
	ignoreReason string, value float64) {
	m.podIdentityAssociations.WithLabelValues(cluster,
		ignoreReason).Set(value)
}

func (m metrics) recordDiscoverLatency(cluster string, elapsed time.Duration) {
	sec := float64(elapsed) / float64(time.Second)
	m.discoverLatency.WithLabelValues(cluster).Set(sec)
}

func (m metrics) recordReconcileLatency(cluster string, elapsed time.Duration) {
	sec := float64(elapsed) / float64(time.Second)
	m.reconcileLatency.WithLabelValues(cluster).Set(sec)
}

func (m metrics) recordAPILatency(cluster, api, status string, elapsed time.Duration) {
	sec := float64(elapsed) / float64(time.Second)
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

func newMetrics(namespace string, latencyBucketsSeconds []float64) metrics {
	registry := prometheus.NewRegistry()

	const subsystem = ""

	return metrics{
		registry: registry,

		serviceAccounts: newGaugeVec(
			registry,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "service_accounts",
				Help:      "Number of Service Accounts.",
			},
			[]string{"cluster", "ignore_reason"},
		),

		podIdentityAssociations: newGaugeVec(
			registry,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "pod_identity_associations",
				Help:      "Number of Pod Identity Associations.",
			},
			[]string{"cluster", "ignore_reason"},
		),

		discoverLatency: newGaugeVec(
			registry,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "discover_latency_seconds",
				Help:      "Latency of discovery.",
			},
			[]string{"cluster"},
		),

		reconcileLatency: newGaugeVec(
			registry,
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "reconcile_latency_seconds",
				Help:      "Latency of reconcile.",
			},
			[]string{"cluster"},
		),

		apiLatency: newHistoryVec(
			registry,
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "api_latency_seconds",
				Help:      "Latency of API calls in seconds.",
				Buckets:   latencyBucketsSeconds,
			},
			[]string{"cluster", "api", "status"},
		),
	}
}

func newGaugeVec(registerer prometheus.Registerer,
	opts prometheus.GaugeOpts,
	labelValues []string) *prometheus.GaugeVec {
	return promauto.With(registerer).NewGaugeVec(opts, labelValues)
}

func newHistoryVec(registerer prometheus.Registerer,
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
