package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps logrus.Logger with additional functionality for O-RAN Near-RT RIC
type Logger struct {
	*logrus.Logger
	config *config.LoggingConfig
}

// ContextKey type for context keys
type ContextKey string

const (
	// ContextKeyRequestID is the key for request ID in context
	ContextKeyRequestID ContextKey = "request_id"
	// ContextKeyE2NodeID is the key for E2 node ID in context
	ContextKeyE2NodeID ContextKey = "e2_node_id"
	// ContextKeyXAppID is the key for xApp ID in context
	ContextKeyXAppID ContextKey = "xapp_id"
	// ContextKeyInterface is the key for interface name in context
	ContextKeyInterface ContextKey = "interface"
)

// Fields represents structured logging fields
type Fields map[string]interface{}

// NewLogger creates a new production-ready logger with O-RAN specific configuration
func NewLogger(cfg *config.LoggingConfig) *Logger {
	logger := logrus.New()

	// Configure log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Configure formatter
	switch strings.ToLower(cfg.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat:   time.RFC3339,
			DisableHTMLEscape: true,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
				logrus.FieldKeyFile:  "file",
			},
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:          true,
			TimestampFormat:        time.RFC3339,
			DisableLevelTruncation: true,
		})
	default:
		// Default to JSON for production
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	}

	// Configure output
	setupOutput(logger, cfg)

	// Add hooks for additional functionality
	setupHooks(logger, cfg)

	return &Logger{
		Logger: logger,
		config: cfg,
	}
}

// setupOutput configures the logger output based on configuration
func setupOutput(logger *logrus.Logger, cfg *config.LoggingConfig) {
	var writers []io.Writer

	// Always include stdout/stderr
	switch strings.ToLower(cfg.Output) {
	case "stderr":
		writers = append(writers, os.Stderr)
	case "stdout":
		writers = append(writers, os.Stdout)
	default:
		writers = append(writers, os.Stdout)
	}

	// Add file output if configured
	if cfg.File.Enabled && cfg.File.Filename != "" {
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(cfg.File.Filename), 0755); err != nil {
			logger.WithError(err).Error("Failed to create log directory")
		} else {
			// Configure log rotation
			fileWriter := &lumberjack.Logger{
				Filename:   cfg.File.Filename,
				MaxSize:    cfg.File.MaxSize, // MB
				MaxBackups: cfg.File.MaxBackups,
				MaxAge:     cfg.File.MaxAge, // days
				Compress:   cfg.File.Compress,
			}
			writers = append(writers, fileWriter)
		}
	}

	// Set multi-writer output
	if len(writers) > 1 {
		logger.SetOutput(io.MultiWriter(writers...))
	} else if len(writers) == 1 {
		logger.SetOutput(writers[0])
	}
}

// setupHooks adds logging hooks for additional functionality
func setupHooks(logger *logrus.Logger, cfg *config.LoggingConfig) {
	// Add context hook for structured logging
	logger.AddHook(&ContextHook{})

	// Add caller information if requested
	if cfg.Structured {
		logger.SetReportCaller(true)
	}

	// Add remote logging hook if configured
	if cfg.Remote.Enabled && cfg.Remote.Endpoint != "" {
		hook := NewRemoteHook(cfg.Remote.Endpoint, cfg.Remote.Protocol)
		logger.AddHook(hook)
	}
}

// WithContext returns a logger with context information
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.Logger.WithContext(ctx)

	// Add trace information if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		entry = entry.WithFields(logrus.Fields{
			"trace_id": span.SpanContext().TraceID().String(),
			"span_id":  span.SpanContext().SpanID().String(),
		})
	}

	// Add request ID if available
	if requestID := ctx.Value(ContextKeyRequestID); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}

	// Add E2 node ID if available
	if nodeID := ctx.Value(ContextKeyE2NodeID); nodeID != nil {
		entry = entry.WithField("e2_node_id", nodeID)
	}

	// Add xApp ID if available
	if xappID := ctx.Value(ContextKeyXAppID); xappID != nil {
		entry = entry.WithField("xapp_id", xappID)
	}

	// Add interface name if available
	if interfaceName := ctx.Value(ContextKeyInterface); interfaceName != nil {
		entry = entry.WithField("interface", interfaceName)
	}

	return entry
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields Fields) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields(fields))
}

// WithField returns a logger with an additional field
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithError returns a logger with error information
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// WithComponent returns a logger with component information
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.Logger.WithField("component", component)
}

// WithInterface returns a logger with O-RAN interface information
func (l *Logger) WithInterface(interfaceName string) *logrus.Entry {
	return l.Logger.WithField("interface", interfaceName)
}

// WithE2Node returns a logger with E2 node information
func (l *Logger) WithE2Node(nodeID string) *logrus.Entry {
	return l.Logger.WithField("e2_node_id", nodeID)
}

// WithXApp returns a logger with xApp information
func (l *Logger) WithXApp(xappID string) *logrus.Entry {
	return l.Logger.WithField("xapp_id", xappID)
}

// LogE2Message logs E2 interface messages with structured information
func (l *Logger) LogE2Message(direction string, nodeID string, messageType string, size int, latency time.Duration) {
	l.WithFields(Fields{
		"interface":    "e2",
		"direction":    direction,
		"e2_node_id":   nodeID,
		"message_type": messageType,
		"message_size": size,
		"latency_ms":   latency.Milliseconds(),
	}).Info("E2 message processed")
}

// LogA1Request logs A1 interface requests with structured information
func (l *Logger) LogA1Request(method string, path string, statusCode int, latency time.Duration, userID string) {
	entry := l.WithFields(Fields{
		"interface":   "a1",
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"latency_ms":  latency.Milliseconds(),
	})

	if userID != "" {
		entry = entry.WithField("user_id", userID)
	}

	if statusCode >= 400 {
		entry.Warn("A1 request failed")
	} else {
		entry.Info("A1 request processed")
	}
}

// LogO1Operation logs O1 interface operations with structured information
func (l *Logger) LogO1Operation(operation string, target string, success bool, latency time.Duration) {
	entry := l.WithFields(Fields{
		"interface":  "o1",
		"operation":  operation,
		"target":     target,
		"success":    success,
		"latency_ms": latency.Milliseconds(),
	})

	if success {
		entry.Info("O1 operation completed")
	} else {
		entry.Warn("O1 operation failed")
	}
}

// LogXAppLifecycle logs xApp lifecycle events with structured information
func (l *Logger) LogXAppLifecycle(xappID string, event string, status string, details map[string]interface{}) {
	fields := Fields{
		"component": "xapp_manager",
		"xapp_id":   xappID,
		"event":     event,
		"status":    status,
	}

	// Add additional details
	for k, v := range details {
		fields[k] = v
	}

	l.WithFields(fields).Info("xApp lifecycle event")
}

// LogSecurityEvent logs security-related events
func (l *Logger) LogSecurityEvent(eventType string, userID string, resource string, action string, success bool, reason string) {
	entry := l.WithFields(Fields{
		"event_type": "security",
		"sub_type":   eventType,
		"user_id":    userID,
		"resource":   resource,
		"action":     action,
		"success":    success,
	})

	if reason != "" {
		entry = entry.WithField("reason", reason)
	}

	if success {
		entry.Info("Security event")
	} else {
		entry.Warn("Security event failed")
	}
}

// LogPerformanceMetric logs performance metrics
func (l *Logger) LogPerformanceMetric(metric string, value float64, unit string, tags map[string]string) {
	fields := Fields{
		"event_type": "performance",
		"metric":     metric,
		"value":      value,
		"unit":       unit,
	}

	// Add tags
	for k, v := range tags {
		fields[fmt.Sprintf("tag_%s", k)] = v
	}

	l.WithFields(fields).Info("Performance metric")
}

// ContextHook adds context information to log entries
type ContextHook struct{}

// Levels returns the log levels for which this hook is executed
func (hook *ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called when a log entry is fired
func (hook *ContextHook) Fire(entry *logrus.Entry) error {
	// Add timestamp in multiple formats for different consumers
	entry.Data["timestamp_unix"] = entry.Time.Unix()
	entry.Data["timestamp_nano"] = entry.Time.UnixNano()

	// Add service information
	entry.Data["service"] = "near-rt-ric"

	// Add hostname
	if hostname, err := os.Hostname(); err == nil {
		entry.Data["hostname"] = hostname
	}

	// Add process ID
	entry.Data["pid"] = os.Getpid()

	return nil
}

// RemoteHook sends log entries to a remote endpoint
type RemoteHook struct {
	endpoint string
	protocol string
}

// NewRemoteHook creates a new remote logging hook
func NewRemoteHook(endpoint, protocol string) *RemoteHook {
	return &RemoteHook{
		endpoint: endpoint,
		protocol: protocol,
	}
}

// Levels returns the log levels for which this hook is executed
func (hook *RemoteHook) Levels() []logrus.Level {
	// Only send warning and above to remote endpoint
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
	}
}

// Fire is called when a log entry is fired
func (hook *RemoteHook) Fire(entry *logrus.Entry) error {
	// Implementation would send log entry to remote endpoint
	// This is a placeholder - in production, you'd implement
	// the actual remote logging protocol (e.g., syslog, ELK, etc.)

	// For now, just add a marker that it would be sent remotely
	entry.Data["remote_logged"] = true

	return nil
}

// GetDefaultLogger returns a default logger for development/testing
func GetDefaultLogger() *Logger {
	cfg := &config.LoggingConfig{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		Structured: true,
		File: config.FileLoggingConfig{
			Enabled: false,
		},
		Remote: config.RemoteLoggingConfig{
			Enabled: false,
		},
	}

	return NewLogger(cfg)
}

// AddContextValue adds a value to the context for logging
func AddContextValue(ctx context.Context, key ContextKey, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// AddRequestID adds a request ID to the context
func AddRequestID(ctx context.Context, requestID string) context.Context {
	return AddContextValue(ctx, ContextKeyRequestID, requestID)
}

// AddE2NodeID adds an E2 node ID to the context
func AddE2NodeID(ctx context.Context, nodeID string) context.Context {
	return AddContextValue(ctx, ContextKeyE2NodeID, nodeID)
}

// AddXAppID adds an xApp ID to the context
func AddXAppID(ctx context.Context, xappID string) context.Context {
	return AddContextValue(ctx, ContextKeyXAppID, xappID)
}

// AddInterface adds an interface name to the context
func AddInterface(ctx context.Context, interfaceName string) context.Context {
	return AddContextValue(ctx, ContextKeyInterface, interfaceName)
}
