package a1

import "time"

// PolicyType represents a type of policy that can be enforced in the Near-RT RIC.
type PolicyType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
}

// PolicyInstance represents an instance of a policy.
type PolicyInstance struct {
	ID         string    `json:"id"`
	TypeID     string    `json:"type_id"`
	Name       string    `json:"name"`
	Parameters string    `json:"parameters"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MLModel represents a machine learning model.
type MLModel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	CreatedAt   time.Time `json:"created_at"`
}

// EnrichmentInfo represents enrichment information.
type EnrichmentInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	DataType  string    `json:"data_type"`
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

// User represents a user of the A1 interface
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// A1NotificationService is a placeholder
type A1NotificationService interface{
	SendNotification(notification *A1Notification) error
	RegisterWebhook(policyID string, webhookURL string) error
	UnregisterWebhook(policyID string) error
	ListWebhooks() (map[string]string, error)
	GetNotificationStatistics() map[string]interface{}
	UpdateConfiguration(retryAttempts int, retryDelay, timeout time.Duration)
	TestWebhook(webhookURL string) error
}

// A1Notification is a placeholder
type A1Notification struct{
	NotificationID   string
	PolicyID         string
	PolicyTypeID     string
	NotificationType A1NotificationType
	Timestamp        time.Time
	Data             map[string]interface{}
}

// A1PolicyType is a placeholder
type A1PolicyType struct {
	PolicyTypeID string                 `json:"policy_type_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	PolicySchema map[string]interface{} `json:"policy_schema"`
	CreatedAt    time.Time              `json:"created_at"`
}

// A1Policy represents an A1 policy instance
type A1Policy struct {
	PolicyID     string                 `json:"policy_id"`
	PolicyTypeID string                 `json:"policy_type_id"`
	PolicyData   map[string]interface{} `json:"policy_data"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	LastModified time.Time              `json:"last_modified"`
}

// PolicyEnforcement is a placeholder
type PolicyEnforcement struct{
	PolicyID string
	Status A1PolicyStatus
	Metrics *PolicyMetrics
	EnforcementTime time.Time
}

// PolicyMetrics is a placeholder
type PolicyMetrics struct{
	EnforcementLatency time.Duration
	SuccessRate float64
	ErrorRate float64
	LastEnforcementTime time.Time
	TotalEnforcements int64
	FailedEnforcements int64
}

// A1ErrorResponse is a placeholder
type A1ErrorResponse struct{
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// TokenResponse represents a token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	SessionID    string `json:"session_id"`
}

// A1Repository defines the interface for A1 data storage and retrieval.
type A1Repository interface {
	// Policy Type operations
	CreatePolicyType(policyType *A1PolicyType) error
	GetPolicyType(policyTypeID string) (*A1PolicyType, error)
	UpdatePolicyType(policyType *A1PolicyType) error
	DeletePolicyType(policyTypeID string) error
	ListPolicyTypes() ([]*A1PolicyType, error)

	// Policy operations
	CreatePolicy(policy *A1Policy) error
	GetPolicy(policyID string) (*A1Policy, error)
	UpdatePolicy(policy *A1Policy) error
	DeletePolicy(policyID string) error
	ListPolicies() ([]*A1Policy, error)
	ListPoliciesByType(policyTypeID string) ([]*A1Policy, error)

	// Policy enforcement tracking
	RecordEnforcement(enforcement *PolicyEnforcement) error
	GetEnforcementStatus(policyID string) (*PolicyEnforcement, error)
	GetPolicyMetrics(policyID string) (*PolicyMetrics, error)

	// Additional utility methods
	GetPolicyTypeStatus(policyTypeID string) (*A1PolicyTypeStatus, error)
	GetStatistics() map[string]interface{}

	// Export/Import functions for backup and restore
	Export() ([]byte, error)
	Import(data []byte) error

	// Cleanup removes old enforcement records and metrics
	Cleanup(maxAge time.Duration) error
}

// A1PolicyTypeStatus is a placeholder
type A1PolicyTypeStatus struct{
	PolicyTypeID string
	NumberOfPolicies int
	PolicyInstanceIDs []string
}

// A1PolicyValidator is a placeholder
type A1PolicyValidator struct{}

// A1NotificationType is a placeholder
type A1NotificationType string

const (
	A1NotificationPolicyCreated A1NotificationType = "POLICY_CREATED"
	A1NotificationPolicyUpdated A1NotificationType = "POLICY_UPDATED"
	A1NotificationPolicyDeleted A1NotificationType = "POLICY_DELETED"
)

// A1PolicyStatus is a placeholder
type A1PolicyStatus string

const (
	A1PolicyStatusEnforced    A1PolicyStatus = "ENFORCED"
	A1PolicyStatusNotEnforced A1PolicyStatus = "NOT_ENFORCED"
	A1PolicyStatusError       A1PolicyStatus = "ERROR"
)
