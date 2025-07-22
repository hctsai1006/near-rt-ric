package e2

import (
	"encoding/asn1"
	"time"
)

// O-RAN E2 Protocol Constants and Types
// Based on O-RAN E2AP v3.0 specification

// E2AP Procedure Codes according to O-RAN.WG3.E2AP-v03.00
const (
	// Basic E2 procedures
	E2SetupRequestID           = 1
	E2SetupResponseID          = 1
	E2SetupFailureID           = 1
	
	// Subscription procedures
	RICSubscriptionRequestID   = 8
	RICSubscriptionResponseID  = 8
	RICSubscriptionFailureID   = 8
	RICSubscriptionDeleteRequestID  = 9
	RICSubscriptionDeleteResponseID = 9
	RICSubscriptionDeleteFailureID  = 9
	
	// Indication procedures
	RICIndicationID            = 5
	
	// Control procedures
	RICControlRequestID        = 4
	RICControlAckID           = 4
	RICControlFailureID       = 4
	
	// Service update procedures
	RICServiceUpdateID         = 7
	RICServiceUpdateAckID      = 7
	RICServiceUpdateFailureID  = 7
	
	// Configuration update procedures
	E2NodeConfigurationUpdateID     = 10
	E2NodeConfigurationUpdateAckID  = 10
	E2NodeConfigurationUpdateFailureID = 10
	
	// Connection update procedures
	E2ConnectionUpdateID       = 11
	E2ConnectionUpdateAckID    = 11
	E2ConnectionUpdateFailureID = 11
	
	// Reset procedures
	ResetRequestID             = 3
	ResetResponseID            = 3
	
	// Error indication
	ErrorIndicationID          = 2
)

// E2AP Criticality values
const (
	CriticalityReject     = 0
	CriticalityIgnore     = 1
	CriticalityNotify     = 2
)

// E2AP Presence values
const (
	PresenceOptional  = 0
	PresenceConditional = 1
	PresenceMandatory = 2
)

// E2NodeType represents the type of E2 node
type E2NodeType string

// E2 Node Types
const (
	E2NodeTypeUnknown E2NodeType = "unknown"
	E2NodeTypeGNB     E2NodeType = "gNB"
	E2NodeTypeENB     E2NodeType = "eNB"
	E2NodeTypeNGENB   E2NodeType = "ng-eNB"
	E2NodeTypeENGNB   E2NodeType = "en-gNB"
)

// RIC Action Types
const (
	RICActionTypeReport  = 0
	RICActionTypeInsert  = 1
	RICActionTypePolicy  = 2
)

// E2 Protocol Constants
const (
	E2SCTPPort = 36421  // O-RAN standard E2 SCTP port
)

// RIC Indication Types
const (
	RICIndicationTypeReport = 0
	RICIndicationTypeInsert = 1
)

// Cause Types
const (
	CauseRadioNetwork     = 0
	CauseTransport        = 1
	CauseProtocol         = 2
	CauseMisc            = 3
	CauseRIC             = 4
)

// Cause Values for Radio Network
const (
	CauseValueUnspecified                = 0
	CauseValueHandoverDesirableForRadioReason = 1
	CauseValueTimeCriticalHandover      = 2
	CauseValueResourceOptimisationHandover = 3
	CauseValueReduceLoadInServingCell   = 4
	CauseValueUserInactivity           = 5
	CauseValueRadioConnectionWithULFailed = 6
	CauseValueFailureInRadioInterfaceProcedure = 7
	CauseValueBearerOptionNotAvailable  = 8
	CauseValueUnknownOrAlreadyAllocatedRICRequestID = 9
	CauseValueNodeRejection            = 10
)

// Node Status
type NodeStatus int

const (
	NodeStatusUnknown NodeStatus = iota
	NodeStatusConnected
	NodeStatusDisconnected
	NodeStatusSetupInProgress
	NodeStatusSetupFailed
	NodeStatusOperational
	NodeStatusMaintenance
	NodeStatusFaulty
)

// Subscription Status
type SubscriptionStatus int

const (
	SubscriptionStatusUnknown SubscriptionStatus = iota
	SubscriptionStatusPending
	SubscriptionStatusActive
	SubscriptionStatusFailed
	SubscriptionStatusDeleted
	SubscriptionStatusExpired
)

// Connection Event Types
type ConnectionEventType int

const (
	ConnectionEstablished ConnectionEventType = iota
	ConnectionClosed
	ConnectionError
	ConnectionTimeout
)

// Note: E2AP PDU structures are defined in asn1.go for proper ASN.1 encoding

// E2APMessage represents a generic E2AP message wrapper
type E2APMessage struct {
	PDU       *E2AP_PDU
	NodeID    string
	Timestamp time.Time
}

// CauseType represents the type of cause in E2AP messages
type CauseType int32

const (
	CauseTypeRIC CauseType = iota
	CauseTypeProtocol
	CauseTypeTransport
	CauseTypeMisc
)

// E2InterfaceConfig contains configuration for E2 interface
type E2InterfaceConfig struct {
	SCTP              SCTPConfig
	Timeout           time.Duration
	MaxNodes          int
	HeartbeatInterval time.Duration
	ConnectionTimeout time.Duration
}

// Global E2 Node ID structures
type GlobalE2NodeID struct {
	GNBNodeID    *GNBID    `asn1:"tag:0,optional"`
	ENBNodeID    *ENBID    `asn1:"tag:1,optional"`
	NGENBNodeID  *NGENBID  `asn1:"tag:2,optional"`
	ENGNBNodeID  *ENGGNBID `asn1:"tag:3,optional"`
}

// gNB Node ID
type GNBID struct {
	PLMNIdentity []byte `asn1:"tag:0"`
	GNBCUUPId    *int64 `asn1:"tag:1,optional"`
	GNBCUCPId    *int64 `asn1:"tag:2,optional"`
	GNBDUId      *int64 `asn1:"tag:3,optional"`
}

// eNB Node ID
type ENBID struct {
	PLMNIdentity []byte `asn1:"tag:0"`
	ENBId        []byte `asn1:"tag:1"`
}

// ng-eNB Node ID
type NGENBID struct {
	PLMNIdentity []byte `asn1:"tag:0"`
	ShortMacroENBId *[]byte `asn1:"tag:1,optional"`
	LongMacroENBId  *[]byte `asn1:"tag:2,optional"`
}

// en-gNB Node ID
type ENGGNBID struct {
	PLMNIdentity []byte `asn1:"tag:0"`
	GNBId        []byte `asn1:"tag:1"`
}

// Global RIC ID
type GlobalRICID struct {
	PLMNIdentity []byte `asn1:"tag:0"`
	RICIdentity  []byte `asn1:"tag:1"`
}

// PLMN Identity (3 bytes)
type PLMNIdentity []byte

// RAN Function structures
type RANFunction struct {
	RANFunctionID          int64  `asn1:"tag:0"`
	RANFunctionDefinition  []byte `asn1:"tag:1"`
	RANFunctionRevision    int64  `asn1:"tag:2"`
	RANFunctionOID         string `asn1:"tag:3,optional"`
}

// RIC Request ID
type RICRequestID struct {
	RICRequestorID int64 `asn1:"tag:0"`
	RICInstanceID  int64 `asn1:"tag:1"`
}

// RIC Subscription Details
type RICSubscriptionDetails struct {
	RICEventTriggerDefinition []byte       `asn1:"tag:0"`
	RICActions                []RICAction  `asn1:"tag:1"`
}

// RIC Action
type RICAction struct {
	RICActionID         int64                    `asn1:"tag:0"`
	RICActionType       asn1.Enumerated          `asn1:"tag:1"`
	RICActionDefinition *[]byte                  `asn1:"tag:2,optional"`
	RICSubsequentAction *RICSubsequentAction     `asn1:"tag:3,optional"`
}

// RIC Subsequent Action
type RICSubsequentAction struct {
	RICSubsequentActionType asn1.Enumerated `asn1:"tag:0"`
	RICTimeToWait          *asn1.Enumerated `asn1:"tag:1,optional"`
}

// Cause structure
type Cause struct {
	RadioNetwork  *asn1.Enumerated `asn1:"tag:0,optional"`
	Transport     *asn1.Enumerated `asn1:"tag:1,optional"`
	Protocol      *asn1.Enumerated `asn1:"tag:2,optional"`
	Misc          *asn1.Enumerated `asn1:"tag:3,optional"`
	RIC           *asn1.Enumerated `asn1:"tag:4,optional"`
}

// Time to Wait values
const (
	TimeToWaitZero    = 0
	TimeToWaitV1s     = 1
	TimeToWaitV2s     = 2
	TimeToWaitV5s     = 3
	TimeToWaitV10s    = 4
	TimeToWaitV20s    = 5
	TimeToWaitV60s    = 6
)

// E2 Setup Request
type E2SetupRequest struct {
	TransactionID           int64                   `json:"transaction_id"`
	GlobalE2NodeID          GlobalE2NodeID          `json:"global_e2_node_id"`
	RANFunctions            []RANFunction           `json:"ran_functions"`
	E2NodeComponentConfigAdd []E2NodeComponentConfig `json:"e2_node_component_config_add,omitempty"`
}

// E2 Setup Response
type E2SetupResponse struct {
	TransactionID            int64                    `json:"transaction_id"`
	GlobalRICID              GlobalRICID              `json:"global_ric_id"`
	RANFunctionsAccepted     []RANFunctionAccepted    `json:"ran_functions_accepted"`
	RANFunctionsRejected     []RANFunctionRejected    `json:"ran_functions_rejected,omitempty"`
	E2NodeComponentConfigUpdateAck []E2NodeComponentConfigUpdateAck `json:"e2_node_component_config_update_ack,omitempty"`
}

// E2 Setup Failure
type E2SetupFailure struct {
	TransactionID          int64                  `json:"transaction_id"`
	Cause                  Cause                  `json:"cause"`
	TimeToWait             *asn1.Enumerated       `json:"time_to_wait,omitempty"`
	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// RAN Function Accepted
type RANFunctionAccepted struct {
	RANFunctionID       int64 `json:"ran_function_id"`
	RANFunctionRevision int64 `json:"ran_function_revision"`
}

// RAN Function Rejected
type RANFunctionRejected struct {
	RANFunctionID int64 `json:"ran_function_id"`
	Cause         Cause `json:"cause"`
}

// E2 Node Component Configuration
type E2NodeComponentConfig struct {
	E2NodeComponentInterfaceType asn1.Enumerated `json:"e2_node_component_interface_type"`
	E2NodeComponentID            interface{}     `json:"e2_node_component_id"`
	E2NodeComponentConfigurationAdd interface{} `json:"e2_node_component_configuration_add"`
}

// E2 Node Component Configuration Update Ack
type E2NodeComponentConfigUpdateAck struct {
	E2NodeComponentInterfaceType asn1.Enumerated `json:"e2_node_component_interface_type"`
	E2NodeComponentID            interface{}     `json:"e2_node_component_id"`
	E2NodeComponentConfigurationAck interface{} `json:"e2_node_component_configuration_ack"`
}

// RIC Subscription Request
type RICSubscriptionRequest struct {
	RICRequestID           RICRequestID           `json:"ric_request_id"`
	RANFunctionID          int64                  `json:"ran_function_id"`
	RICSubscriptionDetails RICSubscriptionDetails `json:"ric_subscription_details"`
}

// RIC Subscription Response
type RICSubscriptionResponse struct {
	RICRequestID         RICRequestID         `json:"ric_request_id"`
	RANFunctionID        int64                `json:"ran_function_id"`
	RICActionAdmitted    []RICActionAdmitted  `json:"ric_action_admitted"`
	RICActionNotAdmitted []RICActionNotAdmitted `json:"ric_action_not_admitted,omitempty"`
}

// RIC Subscription Failure
type RICSubscriptionFailure struct {
	RICRequestID           RICRequestID           `json:"ric_request_id"`
	RANFunctionID          int64                  `json:"ran_function_id"`
	RICActionNotAdmitted   []RICActionNotAdmitted `json:"ric_action_not_admitted"`
	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// RIC Action Admitted
type RICActionAdmitted struct {
	RICActionID int64 `json:"ric_action_id"`
}

// RIC Action Not Admitted
type RICActionNotAdmitted struct {
	RICActionID int64 `json:"ric_action_id"`
	Cause       Cause `json:"cause"`
}

// RIC Subscription Delete Request
type RICSubscriptionDeleteRequest struct {
	RICRequestID  RICRequestID `json:"ric_request_id"`
	RANFunctionID int64        `json:"ran_function_id"`
}

// RIC Subscription Delete Response
type RICSubscriptionDeleteResponse struct {
	RICRequestID  RICRequestID `json:"ric_request_id"`
	RANFunctionID int64        `json:"ran_function_id"`
}

// RIC Subscription Delete Failure
type RICSubscriptionDeleteFailure struct {
	RICRequestID           RICRequestID           `json:"ric_request_id"`
	RANFunctionID          int64                  `json:"ran_function_id"`
	Cause                  Cause                  `json:"cause"`
	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// RIC Indication
type RICIndication struct {
	RICRequestID         RICRequestID        `json:"ric_request_id"`
	RANFunctionID        int64               `json:"ran_function_id"`
	RICActionID          int64               `json:"ric_action_id"`
	RICIndicationSN      *int64              `json:"ric_indication_sn,omitempty"`
	RICIndicationType    asn1.Enumerated     `json:"ric_indication_type"`
	RICIndicationHeader  []byte              `json:"ric_indication_header"`
	RICIndicationMessage []byte              `json:"ric_indication_message"`
	RICCallProcessID     *[]byte             `json:"ric_call_process_id,omitempty"`
}

// RIC Control Request
type RICControlRequest struct {
	RICRequestID         RICRequestID    `json:"ric_request_id"`
	RANFunctionID        int64           `json:"ran_function_id"`
	RICCallProcessID     *[]byte         `json:"ric_call_process_id,omitempty"`
	RICControlHeader     []byte          `json:"ric_control_header"`
	RICControlMessage    []byte          `json:"ric_control_message"`
	RICControlAckRequest *asn1.Enumerated `json:"ric_control_ack_request,omitempty"`
}

// RIC Control Acknowledge
type RICControlAck struct {
	RICRequestID      RICRequestID `json:"ric_request_id"`
	RANFunctionID     int64        `json:"ran_function_id"`
	RICCallProcessID  *[]byte      `json:"ric_call_process_id,omitempty"`
	RICControlOutcome *[]byte      `json:"ric_control_outcome,omitempty"`
}

// RIC Control Failure
type RICControlFailure struct {
	RICRequestID      RICRequestID `json:"ric_request_id"`
	RANFunctionID     int64        `json:"ran_function_id"`
	RICCallProcessID  *[]byte      `json:"ric_call_process_id,omitempty"`
	Cause             Cause        `json:"cause"`
	RICControlOutcome *[]byte      `json:"ric_control_outcome,omitempty"`
}

// Criticality Diagnostics
type CriticalityDiagnostics struct {
	ProcedureCode         *int64                          `json:"procedure_code,omitempty"`
	TriggeringMessage     *asn1.Enumerated                `json:"triggering_message,omitempty"`
	ProcedureCriticality  *asn1.Enumerated                `json:"procedure_criticality,omitempty"`
	IEsCriticalityDiagnostics []CriticalityDiagnosticsIE `json:"ies_criticality_diagnostics,omitempty"`
}

// Criticality Diagnostics IE
type CriticalityDiagnosticsIE struct {
	IECriticality asn1.Enumerated `json:"ie_criticality"`
	IEID          int64           `json:"ie_id"`
	TypeOfError   asn1.Enumerated `json:"type_of_error"`
}

// E2 Node represents a connected E2 node with full state information
type E2Node struct {
	// Identity
	NodeID            string          `json:"node_id"`
	GlobalE2NodeID    GlobalE2NodeID  `json:"global_e2_node_id"`
	
	// Connection details
	ConnectionID      string          `json:"connection_id"`
	RemoteAddress     string          `json:"remote_address"`
	
	// Node information
	NodeType          string          `json:"node_type"`
	RANFunctions      []RANFunction   `json:"ran_functions"`
	SupportedVersions []string        `json:"supported_versions"`
	
	// Status and timing
	Status            NodeStatus      `json:"status"`
	ConnectedAt       time.Time       `json:"connected_at"`
	LastHeartbeat     time.Time       `json:"last_heartbeat"`
	LastActivity      time.Time       `json:"last_activity"`
	
	// Statistics
	MessagesReceived  uint64          `json:"messages_received"`
	MessagesSent      uint64          `json:"messages_sent"`
	ErrorCount        uint64          `json:"error_count"`
	
	// Capabilities
	MaxSubscriptions  int             `json:"max_subscriptions"`
	ActiveSubscriptions int           `json:"active_subscriptions"`
	
	// Configuration
	ConfigVersion     string          `json:"config_version"`
	ComponentConfigs  []E2NodeComponentConfig `json:"component_configs"`
}

// RIC Subscription represents an active subscription
type RICSubscription struct {
	// Identity
	RequestID         RICRequestID           `json:"request_id"`
	SubscriptionID    string                 `json:"subscription_id"`
	
	// Association
	NodeID            string                 `json:"node_id"`
	RANFunctionID     int64                  `json:"ran_function_id"`
	
	// Configuration
	SubscriptionDetails RICSubscriptionDetails `json:"subscription_details"`
	
	// Status and timing
	Status            SubscriptionStatus     `json:"status"`
	CreatedAt         time.Time              `json:"created_at"`
	LastIndication    time.Time              `json:"last_indication"`
	ExpiresAt         *time.Time             `json:"expires_at,omitempty"`
	
	// Statistics
	IndicationsReceived uint64               `json:"indications_received"`
	ErrorCount         uint64                `json:"error_count"`
	
	// Actions
	Actions           []RICAction            `json:"actions"`
	AdmittedActions   []int64                `json:"admitted_actions"`
	RejectedActions   []RICActionNotAdmitted `json:"rejected_actions"`
}

// Connection Event represents an SCTP connection event
type ConnectionEvent struct {
	Type         ConnectionEventType `json:"type"`
	ConnectionID string              `json:"connection_id"`
	NodeID       string              `json:"node_id,omitempty"`
	RemoteAddr   string              `json:"remote_addr"`
	Timestamp    time.Time           `json:"timestamp"`
	Error        error               `json:"error,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// E2 Message represents a complete E2AP message with metadata
type E2Message struct {
	// Identity and routing
	MessageID     string              `json:"message_id"`
	ConnectionID  string              `json:"connection_id"`
	NodeID        string              `json:"node_id"`
	
	// Message content
	Data          []byte              `json:"data"`
	MessageType   E2MessageType       `json:"message_type"`
	ProcedureCode int64               `json:"procedure_code"`
	
	// Timing and processing
	Timestamp     time.Time           `json:"timestamp"`
	ProcessedAt   *time.Time          `json:"processed_at,omitempty"`
	Latency       *time.Duration      `json:"latency,omitempty"`
	
	// Status and error handling
	Status        string              `json:"status"`
	Error         error               `json:"error,omitempty"`
	RetryCount    int                 `json:"retry_count"`
	
	// Context
	RequestID     *RICRequestID       `json:"request_id,omitempty"`
	TransactionID *int64              `json:"transaction_id,omitempty"`
	
	// Metadata
	Size          int                 `json:"size"`
	Compressed    bool                `json:"compressed"`
	Encrypted     bool                `json:"encrypted"`
	Priority      int                 `json:"priority"`
}

// E2MessageType represents the type of E2AP message
type E2MessageType int

const (
	E2SetupRequestMsg E2MessageType = iota
	E2SetupResponseMsg
	E2SetupFailureMsg
	RICSubscriptionRequestMsg
	RICSubscriptionResponseMsg
	RICSubscriptionFailureMsg
	RICSubscriptionDeleteRequestMsg
	RICSubscriptionDeleteResponseMsg
	RICSubscriptionDeleteFailureMsg
	RICIndicationMsg
	RICControlRequestMsg
	RICControlAckMsg
	RICControlFailureMsg
	RICServiceUpdateMsg
	RICServiceUpdateAckMsg
	RICServiceUpdateFailureMsg
	E2NodeConfigurationUpdateMsg
	E2NodeConfigurationUpdateAckMsg
	E2NodeConfigurationUpdateFailureMsg
	E2ConnectionUpdateMsg
	E2ConnectionUpdateAckMsg
	E2ConnectionUpdateFailureMsg
	ResetRequestMsg
	ResetResponseMsg
	ErrorIndicationMsg
	UnknownMsg
)

// String returns the string representation of E2MessageType
func (e E2MessageType) String() string {
	switch e {
	case E2SetupRequestMsg:
		return "E2SetupRequest"
	case E2SetupResponseMsg:
		return "E2SetupResponse"
	case E2SetupFailureMsg:
		return "E2SetupFailure"
	case RICSubscriptionRequestMsg:
		return "RICSubscriptionRequest"
	case RICSubscriptionResponseMsg:
		return "RICSubscriptionResponse"
	case RICSubscriptionFailureMsg:
		return "RICSubscriptionFailure"
	case RICSubscriptionDeleteRequestMsg:
		return "RICSubscriptionDeleteRequest"
	case RICSubscriptionDeleteResponseMsg:
		return "RICSubscriptionDeleteResponse"
	case RICSubscriptionDeleteFailureMsg:
		return "RICSubscriptionDeleteFailure"
	case RICIndicationMsg:
		return "RICIndication"
	case RICControlRequestMsg:
		return "RICControlRequest"
	case RICControlAckMsg:
		return "RICControlAck"
	case RICControlFailureMsg:
		return "RICControlFailure"
	case RICServiceUpdateMsg:
		return "RICServiceUpdate"
	case RICServiceUpdateAckMsg:
		return "RICServiceUpdateAck"
	case RICServiceUpdateFailureMsg:
		return "RICServiceUpdateFailure"
	case E2NodeConfigurationUpdateMsg:
		return "E2NodeConfigurationUpdate"
	case E2NodeConfigurationUpdateAckMsg:
		return "E2NodeConfigurationUpdateAck"
	case E2NodeConfigurationUpdateFailureMsg:
		return "E2NodeConfigurationUpdateFailure"
	case E2ConnectionUpdateMsg:
		return "E2ConnectionUpdate"
	case E2ConnectionUpdateAckMsg:
		return "E2ConnectionUpdateAck"
	case E2ConnectionUpdateFailureMsg:
		return "E2ConnectionUpdateFailure"
	case ResetRequestMsg:
		return "ResetRequest"
	case ResetResponseMsg:
		return "ResetResponse"
	case ErrorIndicationMsg:
		return "ErrorIndication"
	default:
		return "Unknown"
	}
}

// String returns the string representation of NodeStatus
func (n NodeStatus) String() string {
	switch n {
	case NodeStatusUnknown:
		return "Unknown"
	case NodeStatusConnected:
		return "Connected"
	case NodeStatusDisconnected:
		return "Disconnected"
	case NodeStatusSetupInProgress:
		return "SetupInProgress"
	case NodeStatusSetupFailed:
		return "SetupFailed"
	case NodeStatusOperational:
		return "Operational"
	case NodeStatusMaintenance:
		return "Maintenance"
	case NodeStatusFaulty:
		return "Faulty"
	default:
		return "Unknown"
	}
}

// String returns the string representation of SubscriptionStatus
func (s SubscriptionStatus) String() string {
	switch s {
	case SubscriptionStatusUnknown:
		return "Unknown"
	case SubscriptionStatusPending:
		return "Pending"
	case SubscriptionStatusActive:
		return "Active"
	case SubscriptionStatusFailed:
		return "Failed"
	case SubscriptionStatusDeleted:
		return "Deleted"
	case SubscriptionStatusExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// IsActive returns true if the subscription is in an active state
func (s *RICSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive
}

// IsExpired returns true if the subscription has expired
func (s *RICSubscription) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// IsConnected returns true if the node is in a connected state
func (n *E2Node) IsConnected() bool {
	return n.Status == NodeStatusConnected || n.Status == NodeStatusOperational
}

// GetUptime returns the uptime of the node
func (n *E2Node) GetUptime() time.Duration {
	if !n.IsConnected() {
		return 0
	}
	return time.Since(n.ConnectedAt)
}

// GetLastActivityAge returns the time since last activity
func (n *E2Node) GetLastActivityAge() time.Duration {
	return time.Since(n.LastActivity)
}

// IsStale returns true if the node hasn't had activity within the specified duration
func (n *E2Node) IsStale(threshold time.Duration) bool {
	return n.GetLastActivityAge() > threshold
}