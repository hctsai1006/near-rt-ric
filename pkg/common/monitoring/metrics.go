package monitoring

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector handles Prometheus metrics collection
type MetricsCollector struct {
	namespace string
	subsystem string
	A1Metrics *A1Metrics
}

// A1Metrics contains all Prometheus metrics for the A1 interface
type A1Metrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	PolicyOperations *prometheus.CounterVec
	PoliciesActive   prometheus.Gauge
	PolicyErrors     prometheus.Counter
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
	return &MetricsCollector{
		namespace: namespace,
		subsystem: subsystem,
	}
}

// RegisterA1Metrics registers all A1 interface metrics with Prometheus
func (mc *MetricsCollector) RegisterA1Metrics() {
	mc.A1Metrics = &A1Metrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "requests_total",
				Help:      "Total number of A1 API requests.",
			},
			[]string{"method", "path", "code"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "request_duration_seconds",
				Help:      "Histogram of A1 API request latencies.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		PolicyOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "policy_operations_total",
				Help:      "Total number of policy operations.",
			},
			[]string{"operation", "type", "status"},
		),
		PoliciesActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "policies_active_total",
				Help:      "Total number of active policies.",
			},
		),
		PolicyErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "policy_errors_total",
				Help:      "Total number of policy enforcement errors.",
			},
		),
	}

	prometheus.MustRegister(mc.A1Metrics.RequestsTotal)
	prometheus.MustRegister(mc.A1Metrics.RequestDuration)
	prometheus.MustRegister(mc.A1Metrics.PolicyOperations)
	prometheus.MustRegister(mc.A1Metrics.PoliciesActive)
	prometheus.MustRegister(mc.A1Metrics.PolicyErrors)
}

// RecordA1Request records metrics for an A1 API request
func (mc *MetricsCollector) RecordA1Request(method, path string, statusCode int, duration time.Duration, reqSize, respSize int) {
	mc.A1Metrics.RequestsTotal.WithLabelValues(method, path, http.StatusText(statusCode)).Inc()
	mc.A1Metrics.RequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// Handler returns an HTTP handler for the Prometheus metrics endpoint
func (mc *MetricsCollector) Handler() http.Handler {
	return promhttp.Handler()
}
