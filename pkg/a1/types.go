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

// A1NotificationService defines the interface for sending notifications.
type A1NotificationService interface {
    SendNotification(notification *A1Notification) error
    RegisterWebhook(policyID string, webhookURL string) error
    UnregisterWebhook(policyID string) error
    ListWebhooks() (map[string]string, error)
    GetNotificationStatistics() map[string]interface{}
    UpdateConfiguration(retryAttempts int, retryDelay, timeout time.Duration)
    TestWebhook(webhookURL string) error
}

// A1Notification represents a notification sent by the A1 interface.
type A1Notification struct {
    NotificationID   string                 `json:"notification_id"`
    PolicyID         string                 `json:"policy_id"`
    PolicyTypeID     string                 `json:"policy_type_id"`
    NotificationType A1NotificationType     `json:"notification_type"`
    Timestamp        time.Time              `json:"timestamp"`
    Data             map[string]interface{} `json:"data"`
}

// A1PolicyType defines the structure for a policy type.
type A1PolicyType struct {
    PolicyTypeID string                 `json:"policy_type_id"`
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    PolicySchema map[string]interface{} `json:"policy_schema"`
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
}

// A1Policy represents an A1 policy instance.
type A1Policy struct {
    PolicyID     string                 `json:"policy_id"`
    PolicyTypeID string                 `json:"policy_type_id"`
    PolicyData   map[string]interface{} `json:"policy_data"`
    Status       A1PolicyStatus         `json:"status"`
    CreatedAt    time.Time              `json:"created_at"`
    LastModified time.Time              `json:"last_modified"`
}

// PolicyEnforcement represents the enforcement status of a policy.
type PolicyEnforcement struct {
    PolicyID        string         `json:"policy_id"`
    Status          A1PolicyStatus `json:"status"`
    Metrics         *PolicyMetrics `json:"metrics"`
    EnforcementTime time.Time      `json:"enforcement_time"`
    Message         string         `json:"message,omitempty"`
}

// PolicyMetrics contains metrics related to policy enforcement.
type PolicyMetrics struct {
    EnforcementLatency  time.Duration `json:"enforcement_latency"`
    SuccessRate         float64       `json:"success_rate"`
    ErrorRate           float64       `json:"error_rate"`
    LastEnforcementTime time.Time     `json:"last_enforcement_time"`
    TotalEnforcements   int64         `json:"total_enforcements"`
    FailedEnforcements  int64         `json:"failed_enforcements"`
}

// A1ErrorResponse represents a standardized error response.
type A1ErrorResponse struct {
    Type   string `json:"type"`
    Title  string `json:"title"`
    Status int    `json:"status"`
    Detail string `json:"detail"`
}

// TokenResponse represents a token response for OAuth2.
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
    CreatePolicyType(policyType *A1PolicyType) error
    GetPolicyType(policyTypeID string) (*A1PolicyType, error)
    UpdatePolicyType(policyType *A1PolicyType) error
    DeletePolicyType(policyTypeID string) error
    ListPolicyTypes() ([]*A1PolicyType, error)
    CreatePolicy(policy *A1Policy) error
    GetPolicy(policyID string) (*A1Policy, error)
    UpdatePolicy(policy *A1Policy) error
    DeletePolicy(policyID string) error
    ListPolicies() ([]*A1Policy, error)
    ListPoliciesByType(policyTypeID string) ([]*A1Policy, error)
    RecordEnforcement(enforcement *PolicyEnforcement) error
    GetEnforcementStatus(_ string) (*PolicyEnforcement, error)
    GetPolicyMetrics(policyID string) (*PolicyMetrics, error)
    GetPolicyTypeStatus(policyTypeID string) (*A1PolicyTypeStatus, error)
    GetStatistics() (map[string]interface{}, error)
    Export() ([]byte, error)
    Import(data []byte) error
    Cleanup(maxAge time.Duration) error
}

// A1PolicyTypeStatus provides status information for a policy type.
type A1PolicyTypeStatus struct {
    PolicyTypeID      string   `json:"policy_type_id"`
    NumberOfPolicies  int      `json:"number_of_policies"`
    PolicyInstanceIDs []string `json:"policy_instance_ids"`
}

// A1PolicyValidator defines the interface for policy validation.
type A1PolicyValidator interface {
    ValidatePolicyType(policyType *A1PolicyType) error
    ValidatePolicy(policy *A1Policy, policyType *A1PolicyType) error
    ValidateSchema(schema map[string]interface{}) error
}

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