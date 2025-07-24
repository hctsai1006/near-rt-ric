package o1

import (
	"encoding/xml"
	"time"
)



// Notification represents an O1 notification
type Notification struct {
	EventTime time.Time
	Message   string
}

// ConfigOperation represents a configuration operation
type ConfigOperation struct {
	Type   string
	Target string
	Data   string
}

// O1 Interface Types according to O-RAN O1 specification
// Based on O-RAN.WG10.O1-Interface.0-v08.00

// FCAPS operation types (Fault, Configuration, Accounting, Performance, Security)
type OperationType string

const (
	OperationFault         OperationType = "FAULT"
	OperationConfiguration OperationType = "CONFIGURATION"
	OperationAccounting    OperationType = "ACCOUNTING"
	OperationPerformance   OperationType = "PERFORMANCE"
	OperationSecurity      OperationType = "SECURITY"
)

// NETCONF operation types
type NetconfOperation string

const (
	NetconfGet       NetconfOperation = "get"
	NetconfGetConfig NetconfOperation = "get-config"
	NetconfEditConfig NetconfOperation = "edit-config"
	NetconfCopyConfig NetconfOperation = "copy-config"
	NetconfDeleteConfig NetconfOperation = "delete-config"
	NetconfLock       NetconfOperation = "lock"
	NetconfUnlock     NetconfOperation = "unlock"
	NetconfCloseSession NetconfOperation = "close-session"
	NetconfKillSession NetconfOperation = "kill-session"
)

// YANG model identification
type YANGModel struct {
	Name      string `xml:"name" json:"name"`
	Namespace string `xml:"namespace" json:"namespace"`
	Version   string `xml:"version" json:"version"`
	Revision  string `xml:"revision" json:"revision"`
	Features  []string `xml:"feature" json:"features,omitempty"`
}

// NETCONF capabilities
type NetconfCapability struct {
	URI        string            `xml:"uri,attr" json:"uri"`
	Module     string            `xml:"module,attr,omitempty" json:"module,omitempty"`
	Revision   string            `xml:"revision,attr,omitempty" json:"revision,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// NETCONF session information
type NetconfSession struct {
	SessionID    uint32              `xml:"session-id" json:"session_id"`
	Username     string              `xml:"username" json:"username"`
	SourceHost   string              `xml:"source-host" json:"source_host"`
	LoginTime    time.Time           `xml:"login-time" json:"login_time"`
	InRPCs       uint64              `xml:"in-rpcs" json:"in_rpcs"`
	InBadRPCs    uint64              `xml:"in-bad-rpcs" json:"in_bad_rpcs"`
	OutRPCErrors uint64              `xml:"out-rpc-errors" json:"out_rpc_errors"`
	OutNotifications uint64          `xml:"out-notifications" json:"out_notifications"`
	Capabilities []NetconfCapability `xml:"capabilities>capability" json:"capabilities"`
}

// NETCONF datastore types
type DatastoreType string

const (
	DatastoreRunning   DatastoreType = "running"
	DatastoreCandidate DatastoreType = "candidate"
	DatastoreStartup   DatastoreType = "startup"
)

// NETCONF RPC message structure
type NetconfRPC struct {
	XMLName   xml.Name    `xml:"rpc"`
	MessageID string      `xml:"message-id,attr"`
	Operation interface{} `xml:",innerxml"`
}

// NETCONF RPC Reply structure
type NetconfRPCReply struct {
	XMLName   xml.Name `xml:"rpc-reply"`
	MessageID string   `xml:"message-id,attr"`
	Data      interface{} `xml:"data,omitempty"`
	OK        *struct{}   `xml:"ok,omitempty"`
	Error     *NetconfRPCError `xml:"rpc-error,omitempty"`
}

// NETCONF RPC Error structure
type NetconfRPCError struct {
	Type         string `xml:"error-type"`
	Tag          string `xml:"error-tag"`
	Severity     string `xml:"error-severity"`
	AppTag       string `xml:"error-app-tag,omitempty"`
	Path         string `xml:"error-path,omitempty"`
	Message      string `xml:"error-message,omitempty"`
	Info         string `xml:"error-info,omitempty"`
}

// Configuration management structures

// ConfigurationChange represents a configuration change event
type ConfigurationChange struct {
	ChangeID      string                 `json:"change_id"`
	SessionID     uint32                 `json:"session_id"`
	Username      string                 `json:"username"`
	Timestamp     time.Time              `json:"timestamp"`
	Operation     NetconfOperation       `json:"operation"`
	Datastore     DatastoreType          `json:"datastore"`
	Target        string                 `json:"target"`
	Changes       []ConfigurationItem    `json:"changes"`
	Status        string                 `json:"status"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ConfigurationItem represents a single configuration item
type ConfigurationItem struct {
	Path      string      `json:"path"`
	Operation string      `json:"operation"` // create, merge, replace, delete
	OldValue  interface{} `json:"old_value,omitempty"`
	NewValue  interface{} `json:"new_value,omitempty"`
}

// Fault management structures

// AlarmSeverity represents alarm severity levels
type AlarmSeverity string

const (
	AlarmCritical AlarmSeverity = "critical"
	AlarmMajor    AlarmSeverity = "major"
	AlarmMinor    AlarmSeverity = "minor"
	AlarmWarning  AlarmSeverity = "warning"
)

// AlarmStatus represents alarm status
type AlarmStatus string

const (
	AlarmActive      AlarmStatus = "active"
	AlarmAcknowledged AlarmStatus = "acknowledged"
	AlarmCleared     AlarmStatus = "cleared"
)

// Alarm represents a fault alarm
type Alarm struct {
	AlarmID          string                 `json:"alarm_id"`
	AlarmType        string                 `json:"alarm_type"`
	ManagedObject    string                 `json:"managed_object"`
	Source           string                 `json:"source"`
	Severity         AlarmSeverity          `json:"severity"`
	Status           AlarmStatus            `json:"status"`
	ProbableCause    string                 `json:"probable_cause"`
	SpecificProblem  string                 `json:"specific_problem"`
	AdditionalText   string                 `json:"additional_text,omitempty"`
	RaisedTime       time.Time              `json:"raised_time"`
	ClearedTime      *time.Time             `json:"cleared_time,omitempty"`
	AcknowledgedTime *time.Time             `json:"acknowledged_time,omitempty"`
	AcknowledgedBy   string                 `json:"acknowledged_by,omitempty"`
	AdditionalInfo   map[string]interface{} `json:"additional_info,omitempty"`
}

// Performance management structures

// PerformanceMetric represents a performance measurement
type PerformanceMetric struct {
	MetricID        string                 `json:"metric_id"`
	MetricName      string                 `json:"metric_name"`
	ManagedObject   string                 `json:"managed_object"`
	MetricType      string                 `json:"metric_type"` // counter, gauge, histogram
	Value           interface{}            `json:"value"`
	Unit            string                 `json:"unit,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	CollectionTime  time.Time              `json:"collection_time"`
	Granularity     time.Duration          `json:"granularity"`
	Labels          map[string]string      `json:"labels,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceReport represents a collection of performance metrics
type PerformanceReport struct {
	ReportID       string               `json:"report_id"`
	ManagedObject  string               `json:"managed_object"`
	ReportType     string               `json:"report_type"`
	StartTime      time.Time            `json:"start_time"`
	EndTime        time.Time            `json:"end_time"`
	Granularity    time.Duration        `json:"granularity"`
	Metrics        []PerformanceMetric  `json:"metrics"`
	GeneratedTime  time.Time            `json:"generated_time"`
	ReportedBy     string               `json:"reported_by"`
}

// Software management structures

// SoftwareVersion represents software version information
type SoftwareVersion struct {
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	BuildNumber    string            `json:"build_number,omitempty"`
	BuildDate      time.Time         `json:"build_date,omitempty"`
	Vendor         string            `json:"vendor,omitempty"`
	Description    string            `json:"description,omitempty"`
	InstallDate    time.Time         `json:"install_date,omitempty"`
	Status         string            `json:"status"` // active, inactive, corrupted
	Dependencies   []string          `json:"dependencies,omitempty"`
	Checksums      map[string]string `json:"checksums,omitempty"`
}

// SoftwareInventory represents software inventory
type SoftwareInventory struct {
	ManagedObject string            `json:"managed_object"`
	LastUpdate    time.Time         `json:"last_update"`
	Software      []SoftwareVersion `json:"software"`
}

// FileTransfer represents file transfer operations
type FileTransfer struct {
	TransferID    string                 `json:"transfer_id"`
	Operation     string                 `json:"operation"` // upload, download
	LocalPath     string                 `json:"local_path"`
	RemotePath    string                 `json:"remote_path"`
	Protocol      string                 `json:"protocol"` // sftp, scp, http, https
	Status        string                 `json:"status"` // pending, active, completed, failed
	Progress      float64                `json:"progress"` // 0.0 to 1.0
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Size          int64                  `json:"size,omitempty"`
	Checksum      string                 `json:"checksum,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Security management structures

// SecurityPolicy represents a security policy
type SecurityPolicy struct {
	PolicyID     string                 `json:"policy_id"`
	PolicyName   string                 `json:"policy_name"`
	PolicyType   string                 `json:"policy_type"`
	Description  string                 `json:"description"`
	Enabled      bool                   `json:"enabled"`
	Rules        []SecurityRule         `json:"rules"`
	CreatedTime  time.Time              `json:"created_time"`
	UpdatedTime  time.Time              `json:"updated_time"`
	Version      string                 `json:"version"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityRule represents a security rule within a policy
type SecurityRule struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Action      string                 `json:"action"` // allow, deny, log
	Source      string                 `json:"source"`
	Destination string                 `json:"destination"`
	Protocol    string                 `json:"protocol"`
	Port        string                 `json:"port,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Certificate represents a digital certificate
type Certificate struct {
	CertificateID   string    `json:"certificate_id"`
	CommonName      string    `json:"common_name"`
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	SerialNumber    string    `json:"serial_number"`
	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	KeyUsage        []string  `json:"key_usage"`
	ExtKeyUsage     []string  `json:"ext_key_usage,omitempty"`
	SubjectAltNames []string  `json:"subject_alt_names,omitempty"`
	Fingerprint     string    `json:"fingerprint"`
	Status          string    `json:"status"` // valid, expired, revoked
	CertificateData []byte    `json:"certificate_data,omitempty"`
}

// O1 Interface statistics and health
type O1Statistics struct {
	ActiveSessions      int           `json:"active_sessions"`
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	ConfigChanges       int64         `json:"config_changes"`
	ActiveAlarms        int           `json:"active_alarms"`
	TotalAlarms         int64         `json:"total_alarms"`
	LastUpdate          time.Time     `json:"last_update"`
	Uptime              time.Duration `json:"uptime"`
}

// O1HealthCheck represents the health status of O1 interface
type O1HealthCheck struct {
	Status      string            `json:"status"`
	Timestamp   time.Time         `json:"timestamp"`
	Version     string            `json:"version"`
	Components  map[string]string `json:"components"`
	Uptime      time.Duration     `json:"uptime"`
	NetconfPort int               `json:"netconf_port"`
	TLSEnabled  bool              `json:"tls_enabled"`
}

// Event structures for O1 interface

// O1Event represents events in the O1 interface
type O1Event struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	ManagedObject string                 `json:"managed_object"`
	Timestamp     time.Time              `json:"timestamp"`
	Severity      string                 `json:"severity"`
	Source        string                 `json:"source"`
	Description   string                 `json:"description"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// O1 event types
const (
	EventNetconfSessionCreated = "NETCONF_SESSION_CREATED"
	EventNetconfSessionClosed  = "NETCONF_SESSION_CLOSED"
	EventConfigurationChanged  = "CONFIGURATION_CHANGED"
	EventAlarmRaised          = "ALARM_RAISED"
	EventAlarmCleared         = "ALARM_CLEARED"
	EventSoftwareInstalled    = "SOFTWARE_INSTALLED"
	EventSoftwareRemoved      = "SOFTWARE_REMOVED"
	EventFileTransferCompleted = "FILE_TRANSFER_COMPLETED"
	EventSecurityPolicyUpdated = "SECURITY_POLICY_UPDATED"
)

// Request/Response structures for O1 operations

// GetConfigRequest represents a get-config operation request
type GetConfigRequest struct {
	Datastore DatastoreType `xml:"source>datastore" json:"datastore"`
	Filter    string        `xml:"filter,omitempty" json:"filter,omitempty"`
}

// EditConfigRequest represents an edit-config operation request
type EditConfigRequest struct {
	Datastore     DatastoreType `xml:"target>datastore" json:"datastore"`
	DefaultOperation string     `xml:"default-operation,omitempty" json:"default_operation,omitempty"`
	TestOption    string        `xml:"test-option,omitempty" json:"test_option,omitempty"`
	ErrorOption   string        `xml:"error-option,omitempty" json:"error_option,omitempty"`
	Config        string        `xml:"config" json:"config"`
}

// Filter represents a NETCONF filter for get and get-config operations
type Filter struct {
	Type     string `xml:"type,attr,omitempty" json:"type,omitempty"`
	Subtree  string `xml:",innerxml" json:"subtree,omitempty"`
	XPath    string `xml:"select,attr,omitempty" json:"xpath,omitempty"`
}

// Utility methods

// String methods for type safety and debugging
func (ot OperationType) String() string {
	return string(ot)
}

func (no NetconfOperation) String() string {
	return string(no)
}

func (dt DatastoreType) String() string {
	return string(dt)
}

func (as AlarmSeverity) String() string {
	return string(as)
}

func (ast AlarmStatus) String() string {
	return string(ast)
}

// IsValid checks if an operation type is valid
func (ot OperationType) IsValid() bool {
	switch ot {
	case OperationFault, OperationConfiguration, OperationAccounting, OperationPerformance, OperationSecurity:
		return true
	default:
		return false
	}
}

// IsValid checks if a NETCONF operation is valid
func (no NetconfOperation) IsValid() bool {
	switch no {
	case NetconfGet, NetconfGetConfig, NetconfEditConfig, NetconfCopyConfig,
		 NetconfDeleteConfig, NetconfLock, NetconfUnlock, NetconfCloseSession, NetconfKillSession:
		return true
	default:
		return false
	}
}

// IsValid checks if a datastore type is valid
func (dt DatastoreType) IsValid() bool {
	switch dt {
	case DatastoreRunning, DatastoreCandidate, DatastoreStartup:
		return true
	default:
		return false
	}
}

// IsActive checks if an alarm is currently active
func (a *Alarm) IsActive() bool {
	return a.Status == AlarmActive
}

// IsAcknowledged checks if an alarm has been acknowledged
func (a *Alarm) IsAcknowledged() bool {
	return a.Status == AlarmAcknowledged || a.AcknowledgedTime != nil
}

// IsExpired checks if a certificate is expired
func (c *Certificate) IsExpired() bool {
	return time.Now().After(c.NotAfter)
}

// DaysUntilExpiry returns the number of days until certificate expiry
func (c *Certificate) DaysUntilExpiry() int {
	if c.IsExpired() {
		return 0
	}
	duration := c.NotAfter.Sub(time.Now())
	return int(duration.Hours() / 24)
}