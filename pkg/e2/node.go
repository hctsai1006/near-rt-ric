package e2

import (
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
)

// E2Node represents a single E2 node (e.g., a gNB).
type E2Node struct {
	ID             string
	Status         NodeStatus
	LastHeartbeat  time.Time
	GlobalE2NodeID *models.GlobalE2NodeID
	RANFunctions   []*models.RANfunction
}

// NodeStatus represents the status of an E2 node.
type NodeStatus string

const (
	NodeStatusConnected   NodeStatus = "Connected"
	NodeStatusDisconnected NodeStatus = "Disconnected"
	NodeStatusOperational NodeStatus = "Operational"
	NodeStatusFaulty      NodeStatus = "Faulty"
)