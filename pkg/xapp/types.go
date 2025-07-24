package xapp

import (
	"time"
)

// XApp Framework Types according to O-RAN xApp specification
// Based on O-RAN.WG2.xApp-v03.00

// XAppID represents a unique xApp identifier
type XAppID string

// XAppInstanceID represents a unique xApp instance identifier
type XAppInstanceID string

// XAppStatus represents the status of an xApp
type XAppStatus string

const (
	XAppStatusPending   XAppStatus = "PENDING"
	XAppStatusDeploying XAppStatus = "DEPLOYING"
	XAppStatusRunning   XAppStatus = "RUNNING"
	XAppStatusStopping  XAppStatus = "STOPPING"
	XAppStatusStopped   XAppStatus = "STOPPED"
	XAppStatusFailed    XAppStatus = "FAILED"
	XAppStatusUnknown   XAppStatus = "UNKNOWN"
)

// XAppLifecycleState represents lifecycle states
type XAppLifecycleState string

const (
	LifecycleCreated    XAppLifecycleState = "CREATED"
	LifecycleInstalled  XAppLifecycleState = "INSTALLED"
	LifecycleConfigured XAppLifecycleState = "CONFIGURED"
	LifecycleActive     XAppLifecycleState = "ACTIVE"
	LifecycleInactive   XAppLifecycleState = "INACTIVE"
	LifecycleDeleted    XAppLifecycleState = "DELETED"
)

// XAppType represents different types of xApps
type XAppType string

const (
	XAppTypeControl      XAppType = "CONTROL"
	XAppTypeOptimizer    XAppType = "OPTIMIZER"
	XAppTypeAnalytics    XAppType = "ANALYTICS"
	XAppTypeML           XAppType = "ML"
	XAppTypeOrchestrator XAppType = "ORCHESTRATOR"
)

// XApp represents an xApp definition
type XApp struct {
	XAppID      XAppID                 `json:"xapp_id" yaml:"xapp_id"`
	Name        string                 `json:"name" yaml:"name"`
	Version     string                 `json:"version" yaml:"version"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Type        XAppType               `json:"type" yaml:"type"`
	Vendor      string                 `json:"vendor,omitempty" yaml:"vendor,omitempty"`
	License     string                 `json:"license,omitempty" yaml:"license,omitempty"`
	CreatedAt   time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" yaml:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Deployment specification
	Deployment XAppDeploymentSpec `json:"deployment" yaml:"deployment"`

	// Configuration schema
	ConfigSchema interface{} `json:"config_schema,omitempty" yaml:"config_schema,omitempty"`

	// Resource requirements
	Resources XAppResourceRequirements `json:"resources" yaml:"resources"`

	// Service interfaces
	Interfaces XAppInterfaces `json:"interfaces" yaml:"interfaces"`

	// Dependencies
	Dependencies []XAppDependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// XAppInstance represents a running instance of an xApp
type XAppInstance struct {
	InstanceID     XAppInstanceID     `json:"instance_id" yaml:"instance_id"`
	XAppID         XAppID             `json:"xapp_id" yaml:"xapp_id"`
	Name           string             `json:"name" yaml:"name"`
	Status         XAppStatus         `json:"status" yaml:"status"`
	LifecycleState XAppLifecycleState `json:"lifecycle_state" yaml:"lifecycle_state"`
	CreatedAt      time.Time          `json:"created_at" yaml:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" yaml:"updated_at"`
	StartedAt      *time.Time         `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	StoppedAt      *time.Time         `json:"stopped_at,omitempty" yaml:"stopped_at,omitempty"`

	// Runtime configuration
	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`

	// Runtime information
	RuntimeInfo XAppRuntimeInfo `json:"runtime_info" yaml:"runtime_info"`

	// Health status
	HealthStatus XAppHealthStatus `json:"health_status" yaml:"health_status"`

	// Metrics
	Metrics XAppMetrics `json:"metrics" yaml:"metrics"`

	// Error information
	LastError  string `json:"last_error,omitempty" yaml:"last_error,omitempty"`
	ErrorCount int    `json:"error_count" yaml:"error_count"`

	// Deployment target
	DeploymentTarget string `json:"deployment_target,omitempty" yaml:"deployment_target,omitempty"`
}

// XAppDeploymentSpec defines how an xApp should be deployed
type XAppDeploymentSpec struct {
	// Container image
	Image           string `json:"image" yaml:"image"`
	ImagePullPolicy string `json:"image_pull_policy,omitempty" yaml:"image_pull_policy,omitempty"`

	// Replica configuration
	Replicas int32 `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	// Environment variables
	Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

	// Command and arguments
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`
	Args    []string `json:"args,omitempty" yaml:"args,omitempty"`

	// Ports
	Ports []Port `json:"ports,omitempty" yaml:"ports,omitempty"`

	// Volume mounts
	VolumeMounts []VolumeMount `json:"volume_mounts,omitempty" yaml:"volume_mounts,omitempty"`

	// Security context
	SecurityContext *SecurityContext `json:"security_context,omitempty" yaml:"security_context,omitempty"`

	// Health checks
	ReadinessProbe *Probe `json:"readiness_probe,omitempty" yaml:"readiness_probe,omitempty"`
	LivenessProbe  *Probe `json:"liveness_probe,omitempty" yaml:"liveness_probe,omitempty"`

	// Deployment strategy
	Strategy DeploymentStrategy `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// XAppResourceRequirements defines resource requirements and limits
type XAppResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty" yaml:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty" yaml:"limits,omitempty"`
}

// ResourceList defines resource quantities
type ResourceList struct {
	CPU     string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory  string `json:"memory,omitempty" yaml:"memory,omitempty"`
	Storage string `json:"storage,omitempty" yaml:"storage,omitempty"`
	GPU     string `json:"gpu,omitempty" yaml:"gpu,omitempty"`
}

// XAppInterfaces defines the interfaces an xApp provides/requires
type XAppInterfaces struct {
	// REST API endpoints
	REST []RESTInterface `json:"rest,omitempty" yaml:"rest,omitempty"`

	// Message queue interfaces
	MessageQueue []MessageQueueInterface `json:"message_queue,omitempty" yaml:"message_queue,omitempty"`

	// Database interfaces
	Database []DatabaseInterface `json:"database,omitempty" yaml:"database,omitempty"`

	// Custom interfaces
	Custom []CustomInterface `json:"custom,omitempty" yaml:"custom,omitempty"`
}

// RESTInterface defines a REST API interface
type RESTInterface struct {
	Name           string `json:"name" yaml:"name"`
	Port           int32  `json:"port" yaml:"port"`
	Path           string `json:"path,omitempty" yaml:"path,omitempty"`
	Protocol       string `json:"protocol,omitempty" yaml:"protocol,omitempty"` // http, https
	Authentication string `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	Documentation  string `json:"documentation,omitempty" yaml:"documentation,omitempty"`
}

// MessageQueueInterface defines a message queue interface
type MessageQueueInterface struct {
	Name   string   `json:"name" yaml:"name"`
	Type   string   `json:"type" yaml:"type"` // kafka, redis, rabbitmq
	Topics []string `json:"topics,omitempty" yaml:"topics,omitempty"`
	Queues []string `json:"queues,omitempty" yaml:"queues,omitempty"`
	Role   string   `json:"role" yaml:"role"` // producer, consumer, both
}

// DatabaseInterface defines a database interface
type DatabaseInterface struct {
	Name        string   `json:"name" yaml:"name"`
	Type        string   `json:"type" yaml:"type"` // postgresql, mongodb, influxdb
	Database    string   `json:"database,omitempty" yaml:"database,omitempty"`
	Collections []string `json:"collections,omitempty" yaml:"collections,omitempty"`
	Access      string   `json:"access" yaml:"access"` // read, write, readwrite
}

// CustomInterface defines a custom interface
type CustomInterface struct {
	Name          string                 `json:"name" yaml:"name"`
	Type          string                 `json:"type" yaml:"type"`
	Port          int32                  `json:"port,omitempty" yaml:"port,omitempty"`
	Protocol      string                 `json:"protocol" yaml:"protocol"`
	Configuration map[string]interface{} `json:"configuration,omitempty" yaml:"configuration,omitempty"`
}

// XAppDependency defines a dependency on another xApp or service
type XAppDependency struct {
	Name      string `json:"name" yaml:"name"`
	Type      string `json:"type" yaml:"type"` // xapp, service, database
	Version   string `json:"version,omitempty" yaml:"version,omitempty"`
	Required  bool   `json:"required" yaml:"required"`
	Interface string `json:"interface,omitempty" yaml:"interface,omitempty"`
}

// XAppRuntimeInfo contains runtime information about an xApp instance
type XAppRuntimeInfo struct {
	PodName      string            `json:"pod_name,omitempty" yaml:"pod_name,omitempty"`
	NodeName     string            `json:"node_name,omitempty" yaml:"node_name,omitempty"`
	Namespace    string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ContainerID  string            `json:"container_id,omitempty" yaml:"container_id,omitempty"`
	RestartCount int32             `json:"restart_count" yaml:"restart_count"`
	IPAddress    string            `json:"ip_address,omitempty" yaml:"ip_address,omitempty"`
	Ports        []RuntimePort     `json:"ports,omitempty" yaml:"ports,omitempty"`
	Environment  map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`
}

// RuntimePort represents a port exposed by a running xApp
type RuntimePort struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Port        int32  `json:"port" yaml:"port"`
	Protocol    string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	ExposedPort int32  `json:"exposed_port,omitempty" yaml:"json:"exposed_port,omitempty"`
}

// XAppHealthStatus represents the health status of an xApp
type XAppHealthStatus struct {
	Overall         HealthState         `json:"overall" yaml:"overall"`
	ReadinessCheck  HealthCheck         `json:"readiness_check" yaml:"readiness_check"`
	LivenessCheck   HealthCheck         `json:"liveness_check" yaml:"liveness_check"`
	LastHealthCheck time.Time           `json:"last_health_check" yaml:"last_health_check"`
	HealthHistory   []HealthCheckResult `json:"health_history,omitempty" yaml:"health_history,omitempty"`
}

// HealthState represents overall health state
type HealthState string

const (
	HealthStateHealthy   HealthState = "HEALTHY"
	HealthStateUnhealthy HealthState = "UNHEALTHY"
	HealthStateUnknown   HealthState = "UNKNOWN"
	HealthStateDegraded  HealthState = "DEGRADED"
)

// HealthCheck represents a health check configuration
type HealthCheck struct {
	Type             string   `json:"type" yaml:"type"` // http, tcp, exec
	URL              string   `json:"url,omitempty" yaml:"url,omitempty"`
	Port             int32    `json:"port" yaml:"port"`
	Path             string   `json:"path,omitempty" yaml:"path,omitempty"`
	Command          []string `json:"command,omitempty" yaml:"command,omitempty"`
	IntervalSeconds  int32    `json:"interval_seconds,omitempty" yaml:"interval_seconds,omitempty"`
	TimeoutSeconds   int32    `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	FailureThreshold int32    `json:"failure_threshold,omitempty" yaml:"failure_threshold,omitempty"`
	SuccessThreshold int32    `json:"success_threshold,omitempty" yaml:"success_threshold,omitempty"`
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Timestamp time.Time     `json:"timestamp" yaml:"timestamp"`
	State     HealthState   `json:"state" yaml:"state"`
	Message   string        `json:"message,omitempty" yaml:"message,omitempty"`
	Duration  time.Duration `json:"duration" yaml:"duration"`
}

// XAppMetrics contains metrics for an xApp instance
type XAppMetrics struct {
	CPU         MetricValue            `json:"cpu" yaml:"cpu"`
	Memory      MetricValue            `json:"memory" yaml:"memory"`
	Network     NetworkMetrics         `json:"network" yaml:"network"`
	Storage     StorageMetrics         `json:"storage" yaml:"storage"`
	Custom      map[string]interface{} `json:"custom,omitempty" yaml:"custom,omitempty"`
	LastUpdated time.Time              `json:"last_updated" yaml:"last_updated"`
}

// MetricValue represents a metric value with unit
type MetricValue struct {
	Value     float64   `json:"value" yaml:"value"`
	Unit      string    `json:"unit,omitempty" yaml:"unit,omitempty"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// NetworkMetrics contains network-related metrics
type NetworkMetrics struct {
	BytesReceived   int64 `json:"bytes_received" yaml:"bytes_received"`
	BytesSent       int64 `json:"bytes_sent" yaml:"bytes_sent"`
	PacketsReceived int64 `json:"packets_received" yaml:"packets_received"`
	PacketsSent     int64 `json:"packets_sent" yaml:"packets_sent"`
	Errors          int64 `json:"errors" yaml:"errors"`
	Drops           int64 `json:"drops" yaml:"drops"`
}

// StorageMetrics contains storage-related metrics
type StorageMetrics struct {
	Used       int64 `json:"used" yaml:"used"`
	Available  int64 `json:"available" yaml:"available"`
	Total      int64 `json:"total" yaml:"total"`
	ReadOps    int64 `json:"read_ops" yaml:"read_ops"`
	WriteOps   int64 `json:"write_ops" yaml:"write_ops"`
	ReadBytes  int64 `json:"read_bytes" yaml:"read_bytes"`
	WriteBytes int64 `json:"write_bytes" yaml:"write_bytes"`
}

// Supporting types

// EnvVar represents an environment variable
type EnvVar struct {
	Name      string        `json:"name" yaml:"name"`
	Value     string        `json:"value,omitempty" yaml:"value,omitempty"`
	ValueFrom *EnvVarSource `json:"value_from,omitempty" yaml:"value_from,omitempty"`
}

// EnvVarSource represents a source for environment variable value
type EnvVarSource struct {
	ConfigMapKeyRef *ConfigMapKeySelector `json:"config_map_key_ref,omitempty" yaml:"config_map_key_ref,omitempty"`
	SecretKeyRef    *SecretKeySelector    `json:"secret_key_ref,omitempty" yaml:"secret_key_ref,omitempty"`
	FieldRef        *FieldSelector        `json:"field_ref,omitempty" yaml:"field_ref,omitempty"`
}

// ConfigMapKeySelector selects a key from a ConfigMap
type ConfigMapKeySelector struct {
	Name     string `json:"name" yaml:"name"`
	Key      string `json:"key" yaml:"key"`
	Optional bool   `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// SecretKeySelector selects a key from a Secret
type SecretKeySelector struct {
	Name     string `json:"name" yaml:"name"`
	Key      string `json:"key" yaml:"key"`
	Optional bool   `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// FieldSelector selects a field from the pod
type FieldSelector struct {
	FieldPath string `json:"field_path" yaml:"field_path"`
}

// Port represents a port in a container
type Port struct {
	Name          string `json:"name,omitempty" yaml:"name,omitempty"`
	ContainerPort int32  `json:"container_port" yaml:"container_port"`
	Protocol      string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	HostPort      int32  `json:"host_port,omitempty" yaml:"host_port,omitempty"`
}

// VolumeMount describes a mounting of a Volume within a container
type VolumeMount struct {
	Name      string `json:"name" yaml:"name"`
	MountPath string `json:"mount_path" yaml:"mount_path"`
	SubPath   string `json:"sub_path,omitempty" yaml:"sub_path,omitempty"`
	ReadOnly  bool   `json:"read_only,omitempty" yaml:"read_only,omitempty"`
}

// SecurityContext holds security configuration
type SecurityContext struct {
	RunAsUser      *int64        `json:"run_as_user,omitempty" yaml:"run_as_user,omitempty"`
	RunAsGroup     *int64        `json:"run_as_group,omitempty" yaml:"run_as_group,omitempty"`
	RunAsNonRoot   *bool         `json:"run_as_non_root,omitempty" yaml:"run_as_non_root,omitempty"`
	ReadOnlyRootFS bool          `json:"read_only_root_fs,omitempty" yaml:"read_only_root_fs,omitempty"`
	Capabilities   *Capabilities `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// Capabilities adds and drops POSIX capabilities
type Capabilities struct {
	Add  []string `json:"add,omitempty" yaml:"add,omitempty"`
	Drop []string `json:"drop,omitempty" yaml:"drop,omitempty"`
}

// Probe describes a health check
type Probe struct {
	HTTPGet             *HTTPGetAction   `json:"http_get,omitempty" yaml:"http_get,omitempty"`
	TCPSocket           *TCPSocketAction `json:"tcp_socket,omitempty" yaml:"tcp_socket,omitempty"`
	Exec                *ExecAction      `json:"exec,omitempty" yaml:"exec,omitempty"`
	InitialDelaySeconds int32            `json:"initial_delay_seconds,omitempty" yaml:"initial_delay_seconds,omitempty"`
	TimeoutSeconds      int32            `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	PeriodSeconds       int32            `json:"period_seconds,omitempty" yaml:"period_seconds,omitempty"`
	SuccessThreshold    int32            `json:"success_threshold,omitempty" yaml:"success_threshold,omitempty"`
	FailureThreshold    int32            `json:"failure_threshold,omitempty" yaml:"failure_threshold,omitempty"`
}

// HTTPGetAction describes an action based on HTTP Get requests
type HTTPGetAction struct {
	Path        string       `json:"path,omitempty" yaml:"path,omitempty"`
	Port        int32        `json:"port" yaml:"port"`
	Host        string       `json:"host,omitempty" yaml:"host,omitempty"`
	Scheme      string       `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	HTTPHeaders []HTTPHeader `json:"http_headers,omitempty" yaml:"http_headers,omitempty"`
}

// TCPSocketAction describes an action based on opening a socket
type TCPSocketAction struct {
	Port int32  `json:"port" yaml:"port"`
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
}

// ExecAction describes a command to execute
type ExecAction struct {
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`
}

// HTTPHeader describes a custom header to use in HTTP probes
type HTTPHeader struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

// DeploymentStrategy describes how to deploy an xApp
type DeploymentStrategy struct {
	Type          string                   `json:"type,omitempty" yaml:"type,omitempty"` // Recreate, RollingUpdate
	RollingUpdate *RollingUpdateDeployment `json:"rolling_update,omitempty" yaml:"rolling_update,omitempty"`
}

// RollingUpdateDeployment describes the rolling update strategy
type RollingUpdateDeployment struct {
	MaxUnavailable *int32 `json:"max_unavailable,omitempty" yaml:"max_unavailable,omitempty"`
	MaxSurge       *int32 `json:"max_surge,omitempty" yaml:"json:"max_surge,omitempty"`
}

// XApp Events

// XAppEvent represents an event in the xApp lifecycle
type XAppEvent struct {
	EventID    string                 `json:"event_id" yaml:"event_id"`
	EventType  string                 `json:"event_type" yaml:"event_type"`
	XAppID     XAppID                 `json:"xapp_id" yaml:"xapp_id"`
	InstanceID XAppInstanceID         `json:"instance_id,omitempty" yaml:"instance_id,omitempty"`
	Timestamp  time.Time              `json:"timestamp" yaml:"timestamp"`
	Source     string                 `json:"source" yaml:"source"`
	Message    string                 `json:"message" yaml:"message"`
	Details    map[string]interface{} `json:"details,omitempty" yaml:"details,omitempty"`
	Severity   string                 `json:"severity" yaml:"severity"`
}

// XApp event types
const (
	EventXAppCreated      = "XAPP_CREATED"
	EventXAppUpdated      = "XAPP_UPDATED"
	EventXAppDeleted      = "XAPP_DELETED"
	EventInstanceCreated  = "INSTANCE_CREATED"
	EventInstanceStarted  = "INSTANCE_STARTED"
	EventInstanceStopped  = "INSTANCE_STOPPED"
	EventInstanceFailed   = "INSTANCE_FAILED"
	EventInstanceUpdated  = "INSTANCE_UPDATED"
	EventInstanceDeleted  = "INSTANCE_DELETED"
	EventHealthChanged    = "HEALTH_CHANGED"
	EventConfigChanged    = "CONFIG_CHANGED"
	EventResourceWarning  = "RESOURCE_WARNING"
	EventDependencyFailed = "DEPENDENCY_FAILED"
)

// XAppRepository defines the interface for storing and retrieving xApp related data.
type XAppRepository interface {
	SaveDescriptor(descriptor *XAppDescriptor) error
	GetDescriptor(name, version string) (*XAppDescriptor, error)
	DeleteDescriptor(name, version string) error
	ListDescriptors() ([]*XAppDescriptor, error)

	SaveInstance(instance *XAppInstance) error
	GetInstance(id string) (*XAppInstance, error)
	UpdateInstance(instance *XAppInstance) error
	DeleteInstance(id string) error
	ListInstances() ([]*XAppInstance, error)

	SaveEvent(event *XAppEvent) error
	GetEvents(xappID string, limit int) ([]*XAppEvent, error)

	SaveConflict(conflict *XAppConflict) error
	GetConflict(id string) (*XAppConflict, error)
	UpdateConflict(conflict *XAppConflict) error
	ListConflicts() ([]*XAppConflict, error)
}

// XAppOrchestrator defines the interface for deploying and managing xApp instances.
type XAppOrchestrator interface {
	DeployXApp(descriptor *XAppDescriptor, config map[string]interface{}) error
	UndeployXApp(instanceID string) error
	CheckHealth(instanceID string) (*XAppHealth, error)
	GetMetrics(instanceID string) (*XAppMetrics, error)
	StartHealthMonitoring(instanceID string) error
	StopHealthMonitoring(instanceID string) error
}

// XAppRegistry defines the interface for registering and discovering xApp services.
type XAppRegistry interface {
	Register(instance *XAppInstance) error
	Unregister(instanceID string) error
	Discover(query map[string]string) ([]*XAppInstance, error)
}

// XAppFrameworkConfig holds configuration for the xApp framework.
type XAppFrameworkConfig struct {
	ConflictDetection   bool
	HealthCheckInterval time.Duration
	MetricsInterval     time.Duration
	Namespace           string
}

// XAppConflict represents a conflict detected between xApps.
type XAppConflict struct {
	ID           string
	XAppID1      string
	XAppID2      string
	ConflictType ConflictType
	Description  string
	DetectedAt   time.Time
	ResolvedAt   *time.Time
	Status       ConflictStatus
	Resolution   ConflictResolution
}

// ConflictType defines the type of conflict.
type ConflictType string

const (
	ConflictTypeResource  ConflictType = "RESOURCE"
	ConflictTypeInterface ConflictType = "INTERFACE"
	ConflictTypePolicy    ConflictType = "POLICY"
)

// ConflictStatus defines the status of a conflict.
type ConflictStatus string

const (
	ConflictStatusDetected  ConflictStatus = "DETECTED"
	ConflictStatusResolving ConflictStatus = "RESOLVING"
	ConflictStatusResolved  ConflictStatus = "RESOLVED"
	ConflictStatusFailed    ConflictStatus = "FAILED"
)

// ConflictResolution defines the type of resolution applied to a conflict.
type ConflictResolution string

const (
	ResolutionNone        ConflictResolution = "NONE"
	ResolutionPriority    ConflictResolution = "PRIORITY"
	ResolutionNegotiation ConflictResolution = "NEGOTIATION"
	ResolutionTermination ConflictResolution = "TERMINATION"
	ResolutionResource    ConflictResolution = "RESOURCE"
)

// XAppDescriptor represents an xApp descriptor.
type XAppDescriptor struct {
	Name       string
	Version    string
	Image      string
	Namespace  string
	Resources  XAppResourceRequirements
	Interfaces XAppInterfaces
}

// XAppHealth represents the health of an xApp.
type XAppHealth struct {
	Status string
	Checks map[string]HealthCheck
}

// XAppManager defines the interface for managing xApps.
type XAppManager interface {
	Deploy(descriptor *XAppDescriptor, config map[string]interface{}) (*XAppInstance, error)
	Undeploy(xappID string) error
	Start(xappID string) error
	Stop(xappID string) error
	Restart(xappID string) error
	List() ([]*XAppInstance, error)
	Get(xappID string) (*XAppInstance, error)
	GetStatus(xappID string) (XAppStatus, error)
	GetHealth(xappID string) (*XAppHealth, error)
	GetMetrics(xappID string) (*XAppMetrics, error)
	UpdateConfig(xappID string, config map[string]interface{}) error
	GetConfig(xappID string) (map[string]interface{}, error)
	DetectConflicts(xappID string) ([]*XAppConflict, error)
	ResolveConflict(conflictID string, resolution ConflictResolution) error
	GetConflicts() ([]*XAppConflict, error)
	GetEvents(xappID string) ([]*XAppEvent, error)
	Subscribe(eventTypes []XAppEventType) (<-chan *XAppEvent, error)
	Cleanup()
}

// XAppEventType defines the type of an xApp event.
type XAppEventType string

const (
	XAppEventDeployment   XAppEventType = "DEPLOYMENT"
	XAppEventShutdown     XAppEventType = "SHUTDOWN"
	XAppEventStartup      XAppEventType = "STARTUP"
	XAppEventHealthCheck  XAppEventType = "HEALTH_CHECK"
	XAppEventConfigChange XAppEventType = "CONFIG_CHANGE"
	XAppEventError        XAppEventType = "ERROR"
)

// EventSeverity defines the severity of an event.
type EventSeverity string

const (
	EventSeverityInfo     EventSeverity = "INFO"
	EventSeverityWarning  EventSeverity = "WARNING"
	EventSeverityError    EventSeverity = "ERROR"
	EventSeverityCritical EventSeverity = "CRITICAL"
)

// Utility methods

// String methods for type safety and debugging
func (id XAppID) String() string {
	return string(id)
}

func (id XAppInstanceID) String() string {
	return string(id)
}

func (s XAppStatus) String() string {
	return string(s)
}

func (s XAppLifecycleState) String() string {
	return string(s)
}

func (h HealthState) String() string {
	return string(h)
}

// IsValid checks if an xApp status is valid
func (s XAppStatus) IsValid() bool {
	switch s {
	case XAppStatusPending, XAppStatusDeploying, XAppStatusRunning,
		XAppStatusStopping, XAppStatusStopped, XAppStatusFailed, XAppStatusUnknown:
		return true
	default:
		return false
	}
}

// IsRunning checks if an xApp is in a running state
func (s XAppStatus) IsRunning() bool {
	return s == XAppStatusRunning
}

// IsHealthy checks if an xApp is healthy
func (s XAppStatus) IsHealthy() bool {
	return s == XAppStatusRunning
}

// IsFinal checks if an xApp status is final (no more transitions expected)
func (s XAppStatus) IsFinal() bool {
	return s == XAppStatusStopped || s == XAppStatusFailed
}

// IsValid checks if an xApp type is valid
func (t XAppType) IsValid() bool {
	switch t {
	case XAppTypeControl, XAppTypeOptimizer, XAppTypeAnalytics, XAppTypeML, XAppTypeOrchestrator:
		return true
	default:
		return false
	}
}

// IsValid checks if a lifecycle state is valid
func (s XAppLifecycleState) IsValid() bool {
	switch s {
	case LifecycleCreated, LifecycleInstalled, LifecycleConfigured,
		LifecycleActive, LifecycleInactive, LifecycleDeleted:
		return true
	default:
		return false
	}
}

// IsValid checks if a health state is valid
func (h HealthState) IsValid() bool {
	switch h {
	case HealthStateHealthy, HealthStateUnhealthy, HealthStateUnknown, HealthStateDegraded:
		return true
	default:
		return false
	}
}

// GetResourceUsagePercentage calculates resource usage percentage
func (m *XAppMetrics) GetResourceUsagePercentage(requests ResourceList) map[string]float64 {
	usage := make(map[string]float64)

	// CPU usage percentage (assuming CPU value is in cores)
	if requests.CPU != "" {
		// Simplified calculation - in production, parse resource strings properly
		usage["cpu"] = (m.CPU.Value / 1.0) * 100 // Assuming 1 core request
	}

	// Memory usage percentage (assuming memory values are in bytes)
	if requests.Memory != "" {
		// Simplified calculation - in production, parse resource strings properly
		usage["memory"] = (m.Memory.Value / (1024 * 1024 * 1024)) * 100 // Assuming 1GB request
	}

	return usage
}

// HasExceededLimits checks if metrics exceed resource limits
func (m *XAppMetrics) HasExceededLimits(limits ResourceList) bool {
	// Simplified limit checking - in production, implement proper resource parsing
	return false
}

// GetUptime calculates uptime for an xApp instance
func (xi *XAppInstance) GetUptime() time.Duration {
	if xi.StartedAt == nil {
		return 0
	}

	endTime := time.Now()
	if xi.StoppedAt != nil {
		endTime = *xi.StoppedAt
	}

	return endTime.Sub(*xi.StartedAt)
}

// IsHealthy checks if an xApp instance is healthy
func (xi *XAppInstance) IsHealthy() bool {
	return xi.Status.IsHealthy() && xi.HealthStatus.Overall == HealthStateHealthy
}

// GetAge returns the age of an xApp instance
func (xi *XAppInstance) GetAge() time.Duration {
	return time.Since(xi.CreatedAt)
}

// HasDependency checks if an xApp has a specific dependency
func (xa *XApp) HasDependency(name string) bool {
	for _, dep := range xa.Dependencies {
		if dep.Name == name {
			return true
		}
	}
	return false
}

// GetRequiredDependencies returns only required dependencies
func (xa *XApp) GetRequiredDependencies() []XAppDependency {
	var required []XAppDependency
	for _, dep := range xa.Dependencies {
		if dep.Required {
			required = append(required, dep)
		}
	}
	return required
}
