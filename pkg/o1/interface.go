package o1

import "time"

// O1Interface defines the O-RAN O1 interface for managing network functions.
// It provides functionalities for configuration, fault, performance, and security management.
type O1Interface interface {
	// Start brings up the O1 interface, including the NETCONF server.
	Start() error

	// Stop gracefully shuts down the O1 interface.
	Stop() error

	// HealthCheck returns the current health status of the O1 interface.
	HealthCheck() O1HealthCheck

	// GetStatistics returns operational statistics of the O1 interface.
	GetStatistics() O1Statistics

	// SendNotification sends a notification over the O1 interface.
	SendNotification(notification O1Event) error

	// UpdateConfig applies a configuration change to the managed entity.
	UpdateConfig(change ConfigurationChange) error

	// GetConfig retrieves configuration from the managed entity.
	GetConfig(filter Filter) (string, error)

	// GetAlarmList retrieves the current list of active alarms.
	GetAlarmList() ([]Alarm, error)

	// GetPerformanceMetrics retrieves performance metrics.
	GetPerformanceMetrics(filter PerformanceFilter) ([]PerformanceMetric, error)
}

// PerformanceFilter defines criteria for filtering performance metrics.
type PerformanceFilter struct {
	MetricNames   []string
	ManagedObject string
	StartTime     time.Time
	EndTime       time.Time
	Granularity   time.Duration
}
