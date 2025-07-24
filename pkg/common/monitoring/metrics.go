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
	XAppMetrics *XAppMetrics
	O1Metrics *O1Metrics
}

// A1Metrics contains all Prometheus metrics for the A1 interface
type A1Metrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	PolicyOperations *prometheus.CounterVec
	PoliciesActive   prometheus.Gauge
	PolicyErrors     prometheus.Counter
}

// XAppMetrics contains all Prometheus metrics for the xApp framework
type XAppMetrics struct {
	XAppsTotal           *prometheus.GaugeVec
	InstancesTotal       prometheus.Gauge
	InstancesActive      prometheus.Gauge
	InstancesStopped     prometheus.Gauge
	InstancesFailed      prometheus.Gauge
	DeploymentSuccess    prometheus.Counter
	DeploymentErrors     prometheus.Counter
	HealthChecksTotal    *prometheus.CounterVec
}

// O1Metrics contains all Prometheus metrics for the O1 interface
type O1Metrics struct {
	NetconfSessions     prometheus.Gauge
	NetconfMessages     prometheus.Counter
	NetconfOperations   *prometheus.CounterVec
	AlarmsTotal         *prometheus.CounterVec
	PerformanceMetricsTotal *prometheus.CounterVec
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
	mc := &MetricsCollector{
		namespace: namespace,
		subsystem: subsystem,
	}
	mc.RegisterA1Metrics()
	mc.RegisterXAppMetrics()
	mc.RegisterO1Metrics()
	return mc
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

// RegisterXAppMetrics registers all xApp framework metrics with Prometheus
func (mc *MetricsCollector) RegisterXAppMetrics() {
	mc.XAppMetrics = &XAppMetrics{
		XAppsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "xapps_total",
				Help:      "Total number of registered xApps.",
			},
			[]string{"type"},
		),
		InstancesTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "instances_total",
				Help:      "Total number of xApp instances.",
			},
		),
		InstancesActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "instances_active",
				Help:      "Number of active xApp instances.",
			},
		),
		InstancesStopped: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "instances_stopped",
				Help:      "Number of stopped xApp instances.",
			},
		),
		InstancesFailed: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "instances_failed",
				Help:      "Number of failed xApp instances.",
			},
		),
		DeploymentSuccess: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "deployment_success_total",
				Help:      "Total number of successful xApp deployments.",
			},
		),
		DeploymentErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "deployment_errors_total",
				Help:      "Total number of failed xApp deployments.",
			},
		),
		HealthChecksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "health_checks_total",
				Help:      "Total number of xApp health checks.",
			},
			[]string{"status"},
		),
	}

	prometheus.MustRegister(mc.XAppMetrics.XAppsTotal)
	prometheus.MustRegister(mc.XAppMetrics.InstancesTotal)
	prometheus.MustRegister(mc.XAppMetrics.InstancesActive)
	prometheus.MustRegister(mc.XAppMetrics.InstancesStopped)
	prometheus.MustRegister(mc.XAppMetrics.InstancesFailed)
	prometheus.MustRegister(mc.XAppMetrics.DeploymentSuccess)
	prometheus.MustRegister(mc.XAppMetrics.DeploymentErrors)
	prometheus.MustRegister(mc.XAppMetrics.HealthChecksTotal)
}

// RegisterO1Metrics registers all O1 interface metrics with Prometheus
func (mc *MetricsCollector) RegisterO1Metrics() {
	mc.O1Metrics = &O1Metrics{
		NetconfSessions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "netconf_sessions_active",
				Help:      "Number of active NETCONF sessions.",
			},
		),
		NetconfMessages: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "netconf_messages_total",
				Help:      "Total number of NETCONF messages processed.",
			},
		),
		NetconfOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "netconf_operations_total",
				Help:      "Total number of NETCONF operations.",
			},
			[]string{"operation", "status"},
		),
		AlarmsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "alarms_total",
				Help:      "Total number of alarms raised.",
			},
			[]string{"severity", "type"},
		),
		PerformanceMetricsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mc.namespace,
				Subsystem: mc.subsystem,
				Name:      "performance_metrics_total",
				Help:      "Total number of performance metrics collected.",
			},
			[]string{"name", "managed_object", "type"},
		),
	}

	prometheus.MustRegister(mc.O1Metrics.NetconfSessions)
	prometheus.MustRegister(mc.O1Metrics.NetconfMessages)
	prometheus.MustRegister(mc.O1Metrics.NetconfOperations)
	prometheus.MustRegister(mc.O1Metrics.AlarmsTotal)
	prometheus.MustRegister(mc.O1Metrics.PerformanceMetricsTotal)
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