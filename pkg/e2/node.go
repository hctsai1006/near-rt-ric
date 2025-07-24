package e2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// E2NodeManager manages the lifecycle of E2 nodes with O-RAN compliance
type E2NodeManager struct {
	nodes            map[string]*E2Node
	mutex            sync.RWMutex
	logger           *logrus.Logger
	heartbeatTicker  *time.Ticker
	ctx              context.Context
	cancel           context.CancelFunc
	heartbeatInterval time.Duration
	connectionTimeout time.Duration
	maxNodes          int
	nodeUpdateChan    chan NodeUpdate
}

// NodeUpdate represents a node status update
type NodeUpdate struct {
	NodeID    string
	EventType NodeEventType
	Timestamp time.Time
	Details   interface{}
}

// NodeEventType represents the type of node event
type NodeEventType int

const (
	NodeEventConnected NodeEventType = iota
	NodeEventDisconnected
	NodeEventSetupComplete
	NodeEventSetupFailed
	NodeEventHeartbeatMissed
	NodeEventFunctionUpdate
)

// NewE2NodeManager creates a new O-RAN compliant E2NodeManager
func NewE2NodeManager(config *E2InterfaceConfig) *E2NodeManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &E2NodeManager{
		nodes:             make(map[string]*E2Node),
		logger:            logrus.WithField("component", "e2-node-manager"),
		ctx:               ctx,
		cancel:            cancel,
		heartbeatInterval: config.HeartbeatInterval,
		connectionTimeout: config.ConnectionTimeout,
		maxNodes:          config.MaxNodes,
		nodeUpdateChan:    make(chan NodeUpdate, 100),
	}
	
	// Start heartbeat monitoring
	manager.heartbeatTicker = time.NewTicker(manager.heartbeatInterval)
	go manager.heartbeatMonitor()
	
	// Start node update processor
	go manager.processNodeUpdates()
	
	return manager
}

// AddNode adds a new E2 node with proper validation
func (m *E2NodeManager) AddNode(node E2Node) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if maximum node limit reached
	if len(m.nodes) >= m.maxNodes {
		return fmt.Errorf("maximum number of nodes (%d) reached", m.maxNodes)
	}

	// Validate node configuration
	if err := m.validateNode(&node); err != nil {
		return fmt.Errorf("node validation failed: %w", err)
	}

	// Check for duplicate node ID
	if _, exists := m.nodes[node.NodeID]; exists {
		return fmt.Errorf("node with ID %s already exists", node.NodeID)
	}

	// Set initial status
	node.Status = NodeStatusConnected
	node.LastHeartbeat = time.Now()

	// Store the node
	m.nodes[node.NodeID] = &node

	m.logger.WithFields(logrus.Fields{
		"node_id": node.NodeID,
		"node_type": node.NodeType,
		"address": node.RemoteAddress,
		"functions": len(node.RANFunctions),
	}).Info("E2 node added successfully")

	// Send node update notification
	m.sendNodeUpdate(node.NodeID, NodeEventConnected, nil)

	return nil
}

// UpdateNode updates an existing E2 node
func (m *E2NodeManager) UpdateNode(nodeID string, updatedNode E2Node) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node with ID %s not found", nodeID)
	}

	// Update node fields
	node.NodeType = updatedNode.NodeType
	node.GlobalE2NodeID = updatedNode.GlobalE2NodeID
	node.RemoteAddress = updatedNode.RemoteAddress
	node.RANFunctions = updatedNode.RANFunctions
	node.Status = updatedNode.Status
	node.LastHeartbeat = time.Now()
	node.LastActivity = time.Now()

	m.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"status": updatedNode.Status,
		"functions": len(updatedNode.RANFunctions),
	}).Info("E2 node updated")

	return nil
}

// RemoveNode removes an E2 node
func (m *E2NodeManager) RemoveNode(nodeID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node with ID %s not found", nodeID)
	}

	delete(m.nodes, nodeID)

	m.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"node_type": node.NodeType,
	}).Info("E2 node removed")

	// Send node update notification
	m.sendNodeUpdate(nodeID, NodeEventDisconnected, nil)

	return nil
}

// GetNode retrieves a specific E2 node
func (m *E2NodeManager) GetNode(nodeID string) (*E2Node, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node with ID %s not found", nodeID)
	}

	// Return a copy to prevent external modification
	nodeCopy := *node
	return &nodeCopy, nil
}

// GetAllNodes returns all registered E2 nodes
func (m *E2NodeManager) GetAllNodes() map[string]*E2Node {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy of the map to prevent external modification
	nodesCopy := make(map[string]*E2Node)
	for id, node := range m.nodes {
		nodeCopy := *node
		nodesCopy[id] = &nodeCopy
	}

	return nodesCopy
}

// GetNodesByType returns all nodes of a specific type
func (m *E2NodeManager) GetNodesByType(nodeType E2NodeType) []*E2Node {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var nodes []*E2Node
	for _, node := range m.nodes {
		if E2NodeType(node.NodeType) == nodeType {
			nodeCopy := *node
			nodes = append(nodes, &nodeCopy)
		}
	}

	return nodes
}

// GetOperationalNodes returns all nodes in operational status
func (m *E2NodeManager) GetOperationalNodes() []*E2Node {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var nodes []*E2Node
	for _, node := range m.nodes {
		if node.Status == NodeStatusOperational {
			nodeCopy := *node
			nodes = append(nodes, &nodeCopy)
		}
	}

	return nodes
}

// GetNodeStatistics returns statistics about managed nodes
func (m *E2NodeManager) GetNodeStatistics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_nodes"] = len(m.nodes)

	// Count by status
	statusCount := make(map[NodeStatus]int)
	typeCount := make(map[E2NodeType]int)
	operationalCount := 0

	for _, node := range m.nodes {
		statusCount[node.Status]++
		typeCount[E2NodeType(node.NodeType)]++
		if node.Status == NodeStatusOperational {
			operationalCount++
		}
	}

	stats["operational_nodes"] = operationalCount
	stats["status_breakdown"] = statusCount
	stats["type_breakdown"] = typeCount
	stats["max_capacity"] = m.maxNodes
	stats["capacity_usage"] = float64(len(m.nodes)) / float64(m.maxNodes) * 100

	return stats
}

// UpdateHeartbeat updates the heartbeat timestamp for a node
func (m *E2NodeManager) UpdateHeartbeat(nodeID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	node, exists := m.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node with ID %s not found", nodeID)
	}

	node.LastHeartbeat = time.Now()
	return nil
}

// Helper methods

func (m *E2NodeManager) validateNode(node *E2Node) error {
	if node.NodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	// TODO: Validate PLMN Identity from GlobalE2NodeID
	// if len(node.PLMNIdentity) != 3 {
	//     return fmt.Errorf("PLMN Identity must be 3 bytes")
	// }

	if E2NodeType(node.NodeType) == E2NodeTypeUnknown {
		return fmt.Errorf("node type must be specified")
	}

	// TODO: Validate port if needed
	// if node.Port <= 0 || node.Port > 65535 {
	//     return fmt.Errorf("invalid port number: %d", node.Port)
	// }

	return nil
}

func (m *E2NodeManager) sendNodeUpdate(nodeID string, eventType NodeEventType, details interface{}) {
	update := NodeUpdate{
		NodeID:    nodeID,
		EventType: eventType,
		Timestamp: time.Now(),
		Details:   details,
	}

	select {
	case m.nodeUpdateChan <- update:
	default:
		m.logger.Warn("Node update channel full, dropping update")
	}
}

func (m *E2NodeManager) heartbeatMonitor() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.heartbeatTicker.C:
			m.checkHeartbeats()
		}
	}
}

func (m *E2NodeManager) checkHeartbeats() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for nodeID, node := range m.nodes {
		if node.Status == NodeStatusOperational {
			timeSinceHeartbeat := now.Sub(node.LastHeartbeat)
			if timeSinceHeartbeat > m.connectionTimeout {
				m.logger.WithFields(logrus.Fields{
					"node_id": nodeID,
					"time_since_heartbeat": timeSinceHeartbeat,
				}).Warn("Node heartbeat timeout detected")

				node.Status = NodeStatusFaulty
				m.sendNodeUpdate(nodeID, NodeEventHeartbeatMissed, timeSinceHeartbeat)
			}
		}
	}
}

func (m *E2NodeManager) processNodeUpdates() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case update := <-m.nodeUpdateChan:
			m.handleNodeUpdate(update)
		}
	}
}

func (m *E2NodeManager) handleNodeUpdate(update NodeUpdate) {
	m.logger.WithFields(logrus.Fields{
		"node_id": update.NodeID,
		"event_type": update.EventType,
		"timestamp": update.Timestamp,
	}).Info("Processing node update")

	// Here you could implement additional logic such as:
	// - Notifying xApps about node state changes
	// - Updating metrics and monitoring systems
	// - Triggering failover procedures
	// - Logging to audit trail
}

// Cleanup stops the node manager and cleans up resources
func (m *E2NodeManager) Cleanup() {
	m.cancel()
	if m.heartbeatTicker != nil {
		m.heartbeatTicker.Stop()
	}
	close(m.nodeUpdateChan)
	m.logger.Info("E2 Node Manager cleanup completed")
}

