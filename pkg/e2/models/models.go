package models

import (
	"reflect"
	"sync/atomic"
	"time"
)

// ProcedureCode defines the procedure code for E2 messages.
type ProcedureCode int64

const (
	ProcedureCodeE2Setup         ProcedureCode = 1
	ProcedureCodeRICSubscription ProcedureCode = 12
	ProcedureCodeRICControl      ProcedureCode = 13
)

type Criticality int64

const (
	CriticalityReject Criticality = 0
	CriticalityIgnore Criticality = 1
	CriticalityNotify Criticality = 2
)

// RICactionType defines the type of action for a RIC subscription.
type RICactionType int

const (
	Report RICactionType = iota
)

type E2NodeType string

const (
	E2NodeTypeGNB       E2NodeType = "gNB"
	E2NodeTypeUnknown   E2NodeType = "Unknown"
)

type NodeStatus string

const (
	NodeStatusConnected       NodeStatus = "Connected"
	NodeStatusSetupInProgress NodeStatus = "SetupInProgress"
	NodeStatusOperational     NodeStatus = "Operational"
	NodeStatusFaulty          NodeStatus = "Faulty"
)

// GlobalE2NodeID represents the global E2 node ID.
type GlobalE2NodeID struct {
	GNB_ID *GNB_ID
}

// GNB_ID represents the gNB ID.
type GNB_ID struct {
	GNB_ID []byte
}

// RANfunction represents a RAN function.
type RANfunction struct {
	RANfunctionID         int
	RANfunctionDefinition []byte
	RANfunctionRevision   int
}

// E2SetupRequest represents an E2 setup request.
type E2SetupRequest struct {
	TransactionID  int64
	GlobalE2NodeID *GlobalE2NodeID
	RANfunctions   []*RANfunction
}

// RICrequestID represents a RIC request ID.
type RICrequestID struct {
	RICrequestorID int
	RICinstanceID  int
}

// RICaction represents a RIC action.
type RICaction struct {
	RICactionID   int
	RICactionType RICactionType
}

// RICsubscriptionDetails represents the details of a RIC subscription.
type RICsubscriptionDetails struct {
	RICeventTriggerDefinition []byte
	RICactions                []*RICaction
}

// RICSubscriptionRequest represents a RIC subscription request.
type RICSubscriptionRequest struct {
	RICrequestID         *RICrequestID
	RANfunctionID        int
	RICsubscriptionDetails *RICsubscriptionDetails
}

// E2Response is a generic response structure.
type E2Response struct {
	ProcedureCode ProcedureCode
}

// --- Types added to fix compilation ---

type SubscriptionStatus string

const (
	SubscriptionStatusPending SubscriptionStatus = "Pending"
	SubscriptionStatusActive  SubscriptionStatus = "Active"
	SubscriptionStatusFailed  SubscriptionStatus = "Failed"
	SubscriptionStatusDeleted SubscriptionStatus = "Deleted"
	SubscriptionStatusExpired SubscriptionStatus = "Expired"
)

type Cause struct {
	// Dummy
}

type CauseType int

const (
	CauseTypeRICService CauseType = iota
	CauseTypeE2Node
	CauseTypeTransport
	CauseTypeProtocol
	CauseTypeMisc
)

type RICActionNotAdmitted struct {
	RICActionID int64
	Cause       Cause
}

// RICSubscription represents a RIC subscription.
type RICSubscription struct {
	RequestID            RICrequestID
	SubscriptionID       string
	NodeID               string
	RANFunctionID        int
	SubscriptionDetails  RICsubscriptionDetails
	Status               SubscriptionStatus
	CreatedAt            time.Time
	LastIndication       time.Time
	ExpiresAt            *time.Time
	Actions              []*RICaction
	AdmittedActions      []int64
	RejectedActions      []RICActionNotAdmitted
	ErrorCount           int
	IndicationsReceived  int
}

// IsExpired checks if the subscription is expired.
func (s *RICSubscription) IsExpired() bool {
	return s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt)
}

type ConnectionEvent struct {
	Type         ConnectionEventType
	ConnectionID string
	NodeID       string
	RemoteAddr   string
	Timestamp    time.Time
}

type ConnectionEventType int

const (
	ConnectionEstablished ConnectionEventType = iota
	ConnectionClosed
)

type RICcontrolRequest struct {
	// Dummy
}

type E2APMessage struct {
	// Dummy
}

type E2Node struct {
	NodeID           string
	NodeType         string
	GlobalE2NodeID   *GlobalE2NodeID
	RemoteAddress    string
	Status           NodeStatus
	LastHeartbeat    time.Time
	LastActivity     time.Time
	RANFunctions     []*RANfunction
	ConnectedAt      time.Time
}

type E2SetupResponse struct {
	TransactionID        int64
	GlobalRICID          GlobalRICID
	RANFunctionsAccepted []RANFunctionAccepted
}

type GlobalRICID struct {
	PLMNIdentity []byte
	RICIdentity  []byte
}

type RANFunctionAccepted struct {
	RANFunctionID       int
	RANFunctionRevision int
}

type RICSubscriptionResponse struct {
	RICRequestID         RICrequestID
	RICActionAdmitted    []RICActionAdmitted
	RICActionNotAdmitted []RICActionNotAdmitted
}

type RICActionAdmitted struct {
	RICActionID int64
}

type RICIndication struct {
	RICRequestID         RICrequestID
	RANFunctionID        int
	RICActionID          int
	RICIndicationSN      int
	RICIndicationType    int
	RICIndicationMessage []byte
}

type RICControlFailure struct {
	// Dummy
}

type RICControlAck struct {
	// Dummy
}

type RICSubscriptionDeleteRequest struct {
	// Dummy
}

type E2SetupFailure struct {
	// Dummy
}

type E2Message struct {
	MessageID    string
	MessageType  E2MessageType
	NodeID       string
	ConnectionID string
	Data         []byte
}

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
	RICIndicationMsg
	RICControlRequestMsg
	RICControlAckMsg
	RICControlFailureMsg
)

func (m E2MessageType) String() string {
	return [...]string{"E2SetupRequest", "E2SetupResponse", "E2SetupFailure", "RICSubscriptionRequest", "RICSubscriptionResponse", "RICSubscriptionFailure", "RICSubscriptionDeleteRequest", "RICSubscriptionDeleteResponse", "RICIndication", "RICControlRequest", "RICControlAck", "RICControlFailure"}[m]
}

type E2AP_PDU struct {
	InitiatingMessage *InitiatingMessage `asn1:"choice:initiatingMessage,optional"`
	SuccessfulOutcome *SuccessfulOutcome `asn1:"choice:successfulOutcome,optional"`
	UnsuccessfulOutcome *UnsuccessfulOutcome `asn1:"choice:unsuccessfulOutcome,optional"`
}

type InitiatingMessage struct {
	ProcedureCode ProcedureCode `asn1:"value"`
	Criticality   Criticality   `asn1:"value"`
	Value         interface{}   `asn1:"choice:InitiatingMessage"`
}

type SuccessfulOutcome struct {
	ProcedureCode ProcedureCode `asn1:"value"`
	Criticality   Criticality   `asn1:"value"`
	Value         interface{}   `asn1:"choice:SuccessfulOutcome"`
}

type UnsuccessfulOutcome struct {
	ProcedureCode ProcedureCode `asn1:"value"`
	Criticality   Criticality   `asn1:"value"`
	Value         interface{}   `asn1:"choice:UnsuccessfulOutcome"`
}



var E2AP_PDU_TypeMaps = map[string]map[int64]reflect.Type{
	"InitiatingMessage": {
		int64(ProcedureCodeE2Setup):         reflect.TypeOf(E2SetupRequest{}),
		int64(ProcedureCodeRICSubscription): reflect.TypeOf(RICSubscriptionRequest{}),
		int64(ProcedureCodeRICControl):      reflect.TypeOf(RICcontrolRequest{}),
	},
	"SuccessfulOutcome": {
		int64(ProcedureCodeE2Setup):         reflect.TypeOf(E2SetupResponse{}),
		int64(ProcedureCodeRICSubscription): reflect.TypeOf(RICSubscriptionResponse{}),
	},
	"UnsuccessfulOutcome": {
		int64(ProcedureCodeE2Setup):         reflect.TypeOf(E2SetupFailure{}),
		int64(ProcedureCodeRICSubscription): reflect.TypeOf(RICSubscriptionFailure{}),
	},
}

// RICSubscriptionFailure represents a RIC subscription failure.
type RICSubscriptionFailure struct {
	RICRequestID         RICrequestID
	RICActionNotAdmitted []RICActionNotAdmitted
	Cause                Cause
}

// MessageResult represents the result of processing an E2 message.
type MessageResult struct {
	MessageID      string
	WorkerID       int
	ProcessingTime time.Duration
	Response       interface{}
	Error          error
	Success        bool
}

// WorkerPoolStats holds statistics for the worker pool.
type WorkerPoolStats struct {
	TotalMessages     atomic.Uint64
	ProcessedMessages atomic.Uint64
	FailedMessages    atomic.Uint64
	QueueSize         atomic.Int32
	ActiveWorkers     atomic.Int32
	AverageLatency    atomic.Uint64 // Nanoseconds
}

// RICSubscriptionDeleteResponse represents a RIC subscription delete response.
type RICSubscriptionDeleteResponse struct {
	RICRequestID         RICrequestID
	RICActionAdmitted    []RICActionAdmitted
	RICActionNotAdmitted []RICActionNotAdmitted
}
