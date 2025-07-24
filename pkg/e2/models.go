package e2

import (
	"time"
)

// E2APMessage represents a generic E2AP message.
type E2APMessage struct {
	ProcedureCode int
	Criticality   int
	Value         []byte
}

// E2Node represents an E2 node (e.g., gNB).
type E2Node struct {
	NodeID           string
	NodeType         string
	GlobalE2NodeID   GlobalE2NodeID
	RemoteAddress    string
	Status           NodeStatus
	ConnectedAt      time.Time
	LastHeartbeat    time.Time
	RANFunctions     []RANfunction
}

// NodeStatus represents the operational status of an E2 node.
type NodeStatus string

const (
	NodeStatusConnecting      models.NodeStatus = "connecting"
	NodeStatusSetupInProgress models.NodeStatus = "setup-in-progress"
	NodeStatusOperational     models.NodeStatus = "operational"
	NodeStatusDisconnected    models.NodeStatus = "disconnected"
	NodeStatusError           models.NodeStatus = "error"
)

// E2NodeType represents the type of E2 node.
type E2NodeType string

const (
	E2NodeTypeGNB E2NodeType = "gNB"
	E2NodeTypeENB E2NodeType = "eNB"
)

// GlobalE2NodeID is a global E2 node identifier.
type GlobalE2NodeID struct {
	GNB_ID *GNB_ID
	GNBNodeID *GNBID
}

// GNB_ID is a gNodeB identifier.
type GNB_ID struct {
	GNB_ID []byte
}

// GNBID is a gNodeB identifier with PLMNIdentity.
type GNBID struct {
	PLMNIdentity []byte
}

// RANfunction is a RAN function definition.
type RANfunction struct {
	RANfunctionID         int
	RANfunctionDefinition []byte
	RANfunctionRevision   int
}

// E2SetupRequest is an E2 setup request message.
type E2SetupRequest struct {
	TransactionID  int64
	GlobalE2NodeID GlobalE2NodeID
	RANfunctions   []RANfunction
}

// E2SetupResponse is an E2 setup response message.
type E2SetupResponse struct {
	TransactionID        int64
	GlobalRICID          GlobalRICID
	RANFunctionsAccepted []RANFunctionAccepted
}

// GlobalRICID is a global RIC identifier.
type GlobalRICID struct {
	PLMNIdentity []byte
	RICIdentity  []byte
}

// RANFunctionAccepted is a RAN function accepted in E2 setup response.
type RANFunctionAccepted struct {
	RANFunctionID       int
	RANFunctionRevision int
}

// RICSubscriptionRequest is a RIC subscription request message.
type RICSubscriptionRequest struct {
	RICrequestID           RICRequestID
	RANfunctionID          int
	RICsubscriptionDetails RICSubscriptionDetails
}

// RICSubscriptionResponse is a RIC subscription response message.
type RICSubscriptionResponse struct {
	// TODO: Define fields for RIC Subscription Response
}

// RICRequestID is a RIC request identifier.
type RICRequestID struct {
	RICrequestorID int
	RICinstanceID  int
}

// RICSubscriptionDetails contains details of a RIC subscription.
type RICSubscriptionDetails struct {
	RICeventTriggerDefinition []byte
	RICactions                []RICAction
}

// RICAction is a RIC action.
type RICAction struct {
	RICActionID   int
	RICActionType RICActionType
}

// RICActionType represents the type of RIC action.
type RICActionType int

const (
	RICActionTypeReport   RICActionType = 0
	RICActionTypeInsert   RICActionType = 1
	RICActionTypePolicy   RICActionType = 2
)

// RICIndication is a RIC indication message.
type RICIndication struct {
	// TODO: Define fields for RIC Indication
}

// CauseType represents the cause type for E2AP messages.
type CauseType int

const (
	CauseTypeRIC        CauseType = 0
	CauseTypeProtocol   CauseType = 1
	CauseTypeMisc       CauseType = 2
)

// E2AP Procedure Codes
const (
	E2SetupRequestID       = 1
	RICSubscriptionRequestID = 2
	RICIndicationID        = 3
)