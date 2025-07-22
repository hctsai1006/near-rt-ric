package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// MetricsCollector provides comprehensive monitoring for O-RAN Near-RT RIC
type MetricsCollector struct {
	config   *config.MonitoringConfig
	logger   *logrus.Logger
	registry *prometheus.Registry
	server   *http.Server
	
	// O-RAN Interface Metrics
	E2Metrics E2InterfaceMetrics
	A1Metrics A1InterfaceMetrics
	O1Metrics O1InterfaceMetrics
	
	// System Metrics
	SystemMetrics SystemMetrics
	
	// xApp Metrics
	XAppMetrics XAppMetrics
}

// E2InterfaceMetrics contains all E2 interface related metrics
type E2InterfaceMetrics struct {
	// Connection metrics
	ActiveConnections prometheus.Gauge
	TotalConnections  prometheus.Counter
	ConnectionErrors  prometheus.Counter
	
	// Message metrics
	MessagesTotal          prometheus.CounterVec
	MessageSizeBytes       prometheus.HistogramVec
	MessageLatencySeconds  prometheus.HistogramVec
	
	// Node metrics
	RegisteredNodes prometheus.Gauge
	NodeUptime      prometheus.GaugeVec
	
	// Subscription metrics
	ActiveSubscriptions prometheus.Gauge
	SubscriptionErrors  prometheus.Counter
	
	// Service Model metrics
	ServiceModelsActive prometheus.GaugeVec
}

// A1InterfaceMetrics contains all A1 interface related metrics
type A1InterfaceMetrics struct {
	// HTTP request metrics
	RequestsTotal         prometheus.CounterVec
	RequestDurationSeconds prometheus.HistogramVec
	RequestSizeBytes      prometheus.HistogramVec
	ResponseSizeBytes     prometheus.HistogramVec
	
	// Policy metrics
	PoliciesActive    prometheus.Gauge
	PolicyOperations  prometheus.CounterVec
	PolicyErrors      prometheus.Counter
	
	// Authentication metrics
	AuthenticationAttempts prometheus.CounterVec
	AuthenticationLatency  prometheus.Histogram
	
	// Rate limiting metrics
	RateLimitExceeded prometheus.Counter
}

// O1InterfaceMetrics contains all O1 interface related metrics
type O1InterfaceMetrics struct {
	// NETCONF session metrics
	ActiveSessions    prometheus.Gauge
	SessionsTotal     prometheus.Counter
	SessionErrors     prometheus.Counter
	
	// Operation metrics
	OperationsTotal       prometheus.CounterVec
	OperationLatency      prometheus.HistogramVec
	
	// Configuration metrics
	ConfigurationChanges  prometheus.Counter
	ConfigurationErrors   prometheus.Counter
	
	// FCAPS metrics
	FaultEvents       prometheus.CounterVec
	PerformanceMetrics prometheus.GaugeVec
}

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
	// Resource utilization
	CPUUsagePercent    prometheus.Gauge
	MemoryUsageBytes   prometheus.Gauge
	DiskUsageBytes     prometheus.GaugeVec
	
	// Go runtime metrics
	GoroutinesActive   prometheus.Gauge
	HeapSizeBytes      prometheus.Gauge
	GCDurationSeconds  prometheus.Histogram
	
	// Database metrics
	DatabaseConnections prometheus.GaugeVec
	DatabaseOperations  prometheus.CounterVec
	DatabaseLatency     prometheus.HistogramVec
	
	// Redis metrics
	RedisConnections    prometheus.Gauge
	RedisOperations     prometheus.CounterVec
	RedisLatency        prometheus.HistogramVec
}

// XAppMetrics contains xApp framework related metrics
type XAppMetrics struct {
	// Lifecycle metrics
	XAppsActive         prometheus.Gauge
	XAppDeployments     prometheus.CounterVec
	XAppRestarts        prometheus.Counter
	
	// Resource metrics
	XAppCPUUsage        prometheus.GaugeVec
	XAppMemoryUsage     prometheus.GaugeVec
	
	// Communication metrics
	XAppMessages        prometheus.CounterVec
	XAppMessageLatency  prometheus.HistogramVec
	
	// Conflict resolution metrics
	ConflictDetections  prometheus.Counter
	ConflictResolutions prometheus.CounterVec
}

// NewMetricsCollector creates a new metrics collector with O-RAN specific metrics
func NewMetricsCollector(cfg *config.MonitoringConfig, logger *logrus.Logger) *MetricsCollector {
	registry := prometheus.NewRegistry()
	
	collector := &MetricsCollector{
		config:   cfg,
		logger:   logger,
		registry: registry,
	}
	
	// Initialize all metric families
	collector.initializeE2Metrics()
	collector.initializeA1Metrics()
	collector.initializeO1Metrics()
	collector.initializeSystemMetrics()
	collector.initializeXAppMetrics()
	
	// Register all metrics with Prometheus registry
	collector.registerMetrics()
	
	// Set up OpenTelemetry if enabled
	if cfg.OpenTelemetry.Enabled {
		collector.setupOpenTelemetry()
	}
	
	return collector
}

// initializeE2Metrics initializes E2 interface metrics
func (m *MetricsCollector) initializeE2Metrics() {
	namespace := m.config.Prometheus.Namespace
	subsystem := "e2"
	
	m.E2Metrics = E2InterfaceMetrics{
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "active_connections",
			Help:      "Number of active E2 connections",
		}),
		
		TotalConnections: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "connections_total",
			Help:      "Total number of E2 connections established",
		}),
		
		ConnectionErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "connection_errors_total",
			Help:      "Total number of E2 connection errors",
		}),
		
		MessagesTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "messages_total",
				Help:      "Total number of E2 messages processed",
			},
			[]string{"node_id", "message_type", "direction", "status"},
		),
		
		MessageSizeBytes: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "message_size_bytes",
				Help:      "Size of E2 messages in bytes",
				Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536},
			},
			[]string{"message_type", "direction"},
		),
		
		MessageLatencySeconds: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "message_latency_seconds",
				Help:      "Latency of E2 message processing",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"message_type", "operation"},
		),
		
		RegisteredNodes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "registered_nodes",
			Help:      "Number of registered E2 nodes",
		}),
		
		NodeUptime: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "node_uptime_seconds",
				Help:      "Uptime of E2 nodes in seconds",
			},
			[]string{"node_id"},
		),
		
		ActiveSubscriptions: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "active_subscriptions",
			Help:      "Number of active RIC subscriptions",
		}),
		
		SubscriptionErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "subscription_errors_total",
			Help:      "Total number of subscription errors",
		}),
		
		ServiceModelsActive: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "service_models_active",
				Help:      "Number of active E2 service models",
			},
			[]string{"service_model", "version"},
		),
	}
}

// initializeA1Metrics initializes A1 interface metrics
func (m *MetricsCollector) initializeA1Metrics() {
	namespace := m.config.Prometheus.Namespace
	subsystem := "a1"
	
	m.A1Metrics = A1InterfaceMetrics{
		RequestsTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of A1 requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		
		RequestDurationSeconds: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "Duration of A1 requests",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		
		RequestSizeBytes: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_size_bytes",
				Help:      "Size of A1 request bodies",
				Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576},
			},
			[]string{"method", "endpoint"},
		),
		
		ResponseSizeBytes: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "response_size_bytes",
				Help:      "Size of A1 response bodies",
				Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576},
			},
			[]string{"method", "endpoint"},
		),
		
		PoliciesActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "policies_active",
			Help:      "Number of active A1 policies",
		}),
		
		PolicyOperations: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "policy_operations_total",
				Help:      "Total number of policy operations",
			},
			[]string{"operation", "policy_type", "status"},
		),
		
		PolicyErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "policy_errors_total",
			Help:      "Total number of policy errors",
		}),
		
		AuthenticationAttempts: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "authentication_attempts_total",
				Help:      "Total number of authentication attempts",
			},
			[]string{"method", "status"},
		),
		
		AuthenticationLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "authentication_latency_seconds",
			Help:      "Latency of authentication operations",
			Buckets:   prometheus.DefBuckets,
		}),
		
		RateLimitExceeded: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rate_limit_exceeded_total",
			Help:      "Total number of rate limit violations",
		}),
	}
}

// initializeO1Metrics initializes O1 interface metrics
func (m *MetricsCollector) initializeO1Metrics() {
	namespace := m.config.Prometheus.Namespace
	subsystem := "o1"
	
	m.O1Metrics = O1InterfaceMetrics{
		ActiveSessions: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "active_sessions",
			Help:      "Number of active NETCONF sessions",
		}),
		
		SessionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sessions_total",
			Help:      "Total number of NETCONF sessions",
		}),
		
		SessionErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "session_errors_total",
			Help:      "Total number of NETCONF session errors",
		}),
		
		OperationsTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "operations_total",
				Help:      "Total number of NETCONF operations",
			},
			[]string{"operation", "datastore", "status"},
		),
		
		OperationLatency: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "operation_latency_seconds",
				Help:      "Latency of NETCONF operations",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation", "datastore"},
		),
		
		ConfigurationChanges: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "configuration_changes_total",
			Help:      "Total number of configuration changes",
		}),
		
		ConfigurationErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "configuration_errors_total",
			Help:      "Total number of configuration errors",
		}),
		
		FaultEvents: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "fault_events_total",
				Help:      "Total number of fault events",
			},
			[]string{"severity", "component"},
		),
		
		PerformanceMetrics: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "performance_metrics",
				Help:      "Performance metrics from O1 interface",
			},
			[]string{"metric_name", "component"},
		),
	}
}

// initializeSystemMetrics initializes system-level metrics
func (m *MetricsCollector) initializeSystemMetrics() {
	namespace := m.config.Prometheus.Namespace
	subsystem := "system"
	
	m.SystemMetrics = SystemMetrics{
		CPUUsagePercent: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cpu_usage_percent",
			Help:      "CPU usage percentage",
		}),
		
		MemoryUsageBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "memory_usage_bytes",
			Help:      "Memory usage in bytes",
		}),
		
		DiskUsageBytes: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "disk_usage_bytes",
				Help:      "Disk usage in bytes",
			},
			[]string{"mount_point", "device"},
		),
		
		GoroutinesActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "goroutines_active",
			Help:      "Number of active goroutines",
		}),
		
		HeapSizeBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "heap_size_bytes",
			Help:      "Go heap size in bytes",
		}),
		
		GCDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "gc_duration_seconds",
			Help:      "Garbage collection duration",
			Buckets:   prometheus.DefBuckets,
		}),
		
		DatabaseConnections: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "database_connections",
				Help:      "Number of database connections",
			},
			[]string{"database", "state"},
		),
		
		DatabaseOperations: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "database_operations_total",
				Help:      "Total number of database operations",
			},
			[]string{"database", "operation", "status"},
		),
		
		DatabaseLatency: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "database_latency_seconds",
				Help:      "Database operation latency",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"database", "operation"},
		),
		
		RedisConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "redis_connections",
			Help:      "Number of Redis connections",
		}),
		
		RedisOperations: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "redis_operations_total",
				Help:      "Total number of Redis operations",
			},
			[]string{"operation", "status"},
		),
		
		RedisLatency: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "redis_latency_seconds",
				Help:      "Redis operation latency",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
	}
}

// initializeXAppMetrics initializes xApp framework metrics
func (m *MetricsCollector) initializeXAppMetrics() {
	namespace := m.config.Prometheus.Namespace
	subsystem := "xapp"
	
	m.XAppMetrics = XAppMetrics{
		XAppsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "active",
			Help:      "Number of active xApps",
		}),
		
		XAppDeployments: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "deployments_total",
				Help:      "Total number of xApp deployments",
			},
			[]string{"xapp_name", "version", "status"},
		),
		
		XAppRestarts: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "restarts_total",
			Help:      "Total number of xApp restarts",
		}),
		
		XAppCPUUsage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cpu_usage_percent",
				Help:      "CPU usage percentage per xApp",
			},
			[]string{"xapp_name", "xapp_id"},
		),
		
		XAppMemoryUsage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "memory_usage_bytes",
				Help:      "Memory usage in bytes per xApp",
			},
			[]string{"xapp_name", "xapp_id"},
		),
		
		XAppMessages: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "messages_total",
				Help:      "Total number of xApp messages",
			},
			[]string{"xapp_name", "message_type", "direction"},
		),
		
		XAppMessageLatency: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "message_latency_seconds",
				Help:      "xApp message processing latency",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"xapp_name", "message_type"},
		),
		
		ConflictDetections: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "conflict_detections_total",
			Help:      "Total number of conflict detections",
		}),
		
		ConflictResolutions: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "conflict_resolutions_total",
				Help:      "Total number of conflict resolutions",
			},
			[]string{"strategy", "outcome"},
		),
	}
}

// registerMetrics registers all metrics with the Prometheus registry
func (m *MetricsCollector) registerMetrics() {
	// Register E2 metrics
	m.registry.MustRegister(
		m.E2Metrics.ActiveConnections,
		m.E2Metrics.TotalConnections,
		m.E2Metrics.ConnectionErrors,
		&m.E2Metrics.MessagesTotal,
		&m.E2Metrics.MessageSizeBytes,
		&m.E2Metrics.MessageLatencySeconds,
		m.E2Metrics.RegisteredNodes,
		&m.E2Metrics.NodeUptime,
		m.E2Metrics.ActiveSubscriptions,
		m.E2Metrics.SubscriptionErrors,
		&m.E2Metrics.ServiceModelsActive,
	)
	
	// Register A1 metrics
	m.registry.MustRegister(
		&m.A1Metrics.RequestsTotal,
		&m.A1Metrics.RequestDurationSeconds,
		&m.A1Metrics.RequestSizeBytes,
		&m.A1Metrics.ResponseSizeBytes,
		m.A1Metrics.PoliciesActive,
		&m.A1Metrics.PolicyOperations,
		m.A1Metrics.PolicyErrors,
		&m.A1Metrics.AuthenticationAttempts,
		m.A1Metrics.AuthenticationLatency,
		m.A1Metrics.RateLimitExceeded,
	)
	
	// Register O1 metrics
	m.registry.MustRegister(
		m.O1Metrics.ActiveSessions,
		m.O1Metrics.SessionsTotal,
		m.O1Metrics.SessionErrors,
		&m.O1Metrics.OperationsTotal,
		&m.O1Metrics.OperationLatency,
		m.O1Metrics.ConfigurationChanges,
		m.O1Metrics.ConfigurationErrors,
		&m.O1Metrics.FaultEvents,
		&m.O1Metrics.PerformanceMetrics,
	)
	
	// Register system metrics
	m.registry.MustRegister(
		m.SystemMetrics.CPUUsagePercent,
		m.SystemMetrics.MemoryUsageBytes,
		&m.SystemMetrics.DiskUsageBytes,
		m.SystemMetrics.GoroutinesActive,
		m.SystemMetrics.HeapSizeBytes,
		m.SystemMetrics.GCDurationSeconds,
		&m.SystemMetrics.DatabaseConnections,
		&m.SystemMetrics.DatabaseOperations,
		&m.SystemMetrics.DatabaseLatency,
		m.SystemMetrics.RedisConnections,
		&m.SystemMetrics.RedisOperations,
		&m.SystemMetrics.RedisLatency,
	)
	
	// Register xApp metrics
	m.registry.MustRegister(
		m.XAppMetrics.XAppsActive,
		&m.XAppMetrics.XAppDeployments,
		m.XAppMetrics.XAppRestarts,
		&m.XAppMetrics.XAppCPUUsage,
		&m.XAppMetrics.XAppMemoryUsage,
		&m.XAppMetrics.XAppMessages,
		&m.XAppMetrics.XAppMessageLatency,
		m.XAppMetrics.ConflictDetections,
		&m.XAppMetrics.ConflictResolutions,
	)
	
	// Register Go runtime metrics
	m.registry.MustRegister(prometheus.NewGoCollector())
	m.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
}

// setupOpenTelemetry configures OpenTelemetry metrics if enabled
func (m *MetricsCollector) setupOpenTelemetry() {
	// Create a Prometheus exporter
	exporter, err := otelprometheus.New()
	if err != nil {
		m.logger.WithError(err).Error("Failed to create OpenTelemetry Prometheus exporter")
		return
	}
	
	// Create a metric provider
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	
	m.logger.Info("OpenTelemetry metrics configured")
}

// Start starts the metrics HTTP server
func (m *MetricsCollector) Start(ctx context.Context) error {
	if !m.config.Enabled {
		m.logger.Info("Metrics collection disabled")
		return nil
	}
	
	// Create HTTP server for metrics endpoint
	mux := http.NewServeMux()
	mux.Handle(m.config.Path, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	
	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	m.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", m.config.Host, m.config.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	m.logger.WithFields(logrus.Fields{
		"address": m.server.Addr,
		"path":    m.config.Path,
	}).Info("Starting metrics server")
	
	// Start server in goroutine
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.WithError(err).Error("Metrics server error")
		}
	}()
	
	return nil
}

// Stop gracefully stops the metrics server
func (m *MetricsCollector) Stop(ctx context.Context) error {
	if m.server == nil {
		return nil
	}
	
	m.logger.Info("Stopping metrics server")
	
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	return m.server.Shutdown(shutdownCtx)
}

// RecordE2Message records metrics for E2 message processing
func (m *MetricsCollector) RecordE2Message(nodeID, messageType, direction, status string, size int, latency time.Duration) {
	m.E2Metrics.MessagesTotal.WithLabelValues(nodeID, messageType, direction, status).Inc()
	m.E2Metrics.MessageSizeBytes.WithLabelValues(messageType, direction).Observe(float64(size))
	m.E2Metrics.MessageLatencySeconds.WithLabelValues(messageType, "process").Observe(latency.Seconds())
}

// RecordA1Request records metrics for A1 request processing
func (m *MetricsCollector) RecordA1Request(method, endpoint string, statusCode int, duration time.Duration, requestSize, responseSize int) {
	m.A1Metrics.RequestsTotal.WithLabelValues(method, endpoint, fmt.Sprintf("%d", statusCode)).Inc()
	m.A1Metrics.RequestDurationSeconds.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.A1Metrics.RequestSizeBytes.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	m.A1Metrics.ResponseSizeBytes.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordO1Operation records metrics for O1 operation processing
func (m *MetricsCollector) RecordO1Operation(operation, datastore, status string, latency time.Duration) {
	m.O1Metrics.OperationsTotal.WithLabelValues(operation, datastore, status).Inc()
	m.O1Metrics.OperationLatency.WithLabelValues(operation, datastore).Observe(latency.Seconds())
}

// UpdateSystemMetrics updates system-level metrics
func (m *MetricsCollector) UpdateSystemMetrics(cpuPercent float64, memoryBytes uint64, goroutines int) {
	m.SystemMetrics.CPUUsagePercent.Set(cpuPercent)
	m.SystemMetrics.MemoryUsageBytes.Set(float64(memoryBytes))
	m.SystemMetrics.GoroutinesActive.Set(float64(goroutines))
}

// RecordXAppEvent records xApp lifecycle events
func (m *MetricsCollector) RecordXAppEvent(xappName, version, status string) {
	m.XAppMetrics.XAppDeployments.WithLabelValues(xappName, version, status).Inc()
}