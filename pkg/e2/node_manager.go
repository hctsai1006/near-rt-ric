package e2

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
)

// E2NodeManager manages E2 nodes connected to the Near-RT RIC.
type E2NodeManager struct {
	nodes  map[string]models.E2Node
	mutex  sync.RWMutex
	logger *logrus.Logger
}

// NewE2NodeManager creates a new E2NodeManager.
func NewE2NodeManager(logger *logrus.Logger) *E2NodeManager {
	return &E2NodeManager{
		nodes:  make(map[string]models.E2Node),
		logger: logger.WithField("component", "e2-node-manager"),
	}
}

// AddNode adds a new E2 node to the manager.
func (nm *E2NodeManager) AddNode(node models.E2Node) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	nm.nodes[node.NodeID] = node
	nm.logger.WithField("node_id", node.NodeID).Info("E2 node added")
}

// GetNode retrieves an E2 node by its ID.
func (nm *E2NodeManager) GetNode(nodeID string) (models.E2Node, bool) {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	node, exists := nm.nodes[nodeID]
	return node, exists
}

// UpdateNode updates an existing E2 node.
func (nm *E2NodeManager) UpdateNode(nodeID string, updatedNode models.E2Node) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	if _, exists := nm.nodes[nodeID]; exists {
		nm.nodes[nodeID] = updatedNode
		nm.logger.WithField("node_id", nodeID).Info("E2 node updated")
	} else {
		nm.logger.WithField("node_id", nodeID).Warn("Attempted to update non-existent E2 node")
	}
}

// DeleteNode removes an E2 node from the manager.
func (nm *E2NodeManager) DeleteNode(nodeID string) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	if _, exists := nm.nodes[nodeID]; exists {
		delete(nm.nodes, nodeID)
		nm.logger.WithField("node_id", nodeID).Info("E2 node deleted")
	} else {
		nm.logger.WithField("node_id", nodeID).Warn("Attempted to delete non-existent E2 node")
	}
}

// GetAllNodes returns all managed E2 nodes.
func (nm *E2NodeManager) GetAllNodes() []models.E2Node {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	nodes := make([]models.E2Node, 0, len(nm.nodes))
	for _, node := range nm.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// UpdateNodeStatus updates the status of an E2 node.
func (nm *E2NodeManager) UpdateNodeStatus(nodeID string, status models.NodeStatus) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	if node, exists := nm.nodes[nodeID]; exists {
		node.Status = status
		node.LastHeartbeat = time.Now()
		nm.nodes[nodeID] = node
		nm.logger.WithFields(logrus.Fields{
			"node_id": nodeID,
			"status":  status,
		}).Debug("E2 node status updated")
	} else {
		nm.logger.WithField("node_id", nodeID).Warn("Attempted to update status of non-existent E2 node")
	}
}
