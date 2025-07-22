package e2

import (
	"context"
	"encoding/asn1"
	"fmt"
	"sync"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// NodeManager manages E2 node lifecycle and state
// Implements O-RAN.WG3.E2AP-v03.00 node management requirements
type NodeManager struct {
	config  *config.E2Config
	logger  *logrus.Logger
	metrics *monitoring.MetricsCollector
	codec   *ASN1Codec

	// Node storage and management
	nodes      map[string]*E2Node
	nodesMutex sync.RWMutex

	// Event handling
	eventHandlers []NodeEventHandler

	// Background tasks
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Configuration
	setupTimeout      time.Duration
	heartbeatInterval time.Duration
	staleThreshold    time.Duration
}

// NodeEventHandler defines interface for handling node events
type NodeEventHandler interface {
	OnNodeConnected(node *E2Node)
	OnNodeDisconnected(node *E2Node)
	OnNodeSetupCompleted(node *E2Node)
	OnNodeSetupFailed(node *E2Node, reason string)
	OnNodeError(node *E2Node, err error)
}

// NodeEvent represents a node lifecycle event
type NodeEvent struct {
	Type      NodeEventType
	Node      *E2Node
	Timestamp time.Time
	Error     error
	Details   map[string]interface{}
}

// Note: NodeEventType and related constants are defined in node.go

// NewNodeManager creates a new E2 node manager
func NewNodeManager(config *config.E2Config, logger *logrus.Logger, metrics *monitoring.MetricsCollector, codec *ASN1Codec) *NodeManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &NodeManager{
		config:            config,
		logger:            logger.WithField("component", "node-manager").Logger,
		metrics:           metrics,
		codec:             codec,
		nodes:             make(map[string]*E2Node),
		ctx:               ctx,
		cancel:            cancel,
		setupTimeout:      30 * time.Second,
		heartbeatInterval: 60 * time.Second,
		staleThreshold:    300 * time.Second,
	}
}

// Start starts the node manager
func (nm *NodeManager) Start(ctx context.Context) error {
	nm.logger.Info("Starting E2 Node Manager")

	// Start background workers
	nm.wg.Add(2)
	go nm.staleNodeMonitor()
	go nm.statisticsCollector()

	nm.logger.Info("E2 Node Manager started successfully")
	return nil
}

// Stop stops the node manager
func (nm *NodeManager) Stop(ctx context.Context) error {
	nm.logger.Info("Stopping E2 Node Manager")

	// Cancel context to stop background workers
	nm.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		nm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		nm.logger.Info("E2 Node Manager stopped successfully")
	case <-ctx.Done():
		nm.logger.Warn("E2 Node Manager shutdown timeout")
	}

	return nil
}

// RegisterNode registers a new E2 node connection
func (nm *NodeManager) RegisterNode(connectionID, remoteAddr string) (*E2Node, error) {
	nm.nodesMutex.Lock()
	defer nm.nodesMutex.Unlock()

	// Check if node already exists for this connection
	for _, node := range nm.nodes {
		if node.ConnectionID == connectionID {
			return nil, fmt.Errorf("node already registered for connection %s", connectionID)
		}
	}

	// Create new node
	node := &E2Node{
		ConnectionID:        connectionID,
		RemoteAddress:       remoteAddr,
		Status:              NodeStatusConnected,
		ConnectedAt:         time.Now(),
		LastActivity:        time.Now(),
		LastHeartbeat:       time.Now(),
		RANFunctions:        make([]RANFunction, 0),
		SupportedVersions:   make([]string, 0),
		ComponentConfigs:    make([]E2NodeComponentConfig, 0),
		MaxSubscriptions:    100, // Default value
		ActiveSubscriptions: 0,
	}

	// Generate temporary node ID until E2 Setup is completed
	tempNodeID := fmt.Sprintf("temp_%s_%d", connectionID[:8], time.Now().Unix())
	node.NodeID = tempNodeID

	// Store node
	nm.nodes[tempNodeID] = node

	// Update metrics
	nm.metrics.E2Metrics.RegisteredNodes.Inc()

	nm.logger.WithFields(logrus.Fields{
		"node_id":       tempNodeID,
		"connection_id": connectionID,
		"remote_addr":   remoteAddr,
	}).Info("E2 node registered")

	// Send event
	nm.sendEvent(NodeEvent{
		Type:      NodeEventConnected,
		Node:      node,
		Timestamp: time.Now(),
	})

	return node, nil
}

// HandleE2SetupRequest processes an E2 Setup Request from a node
func (nm *NodeManager) HandleE2SetupRequest(connectionID string, req *E2SetupRequest) error {
	nm.nodesMutex.Lock()
	defer nm.nodesMutex.Unlock()

	// Find node by connection ID
	var node *E2Node
	var tempNodeID string
	for id, n := range nm.nodes {
		if n.ConnectionID == connectionID {
			node = n
			tempNodeID = id
			break
		}
	}

	if node == nil {
		return fmt.Errorf("no node found for connection %s", connectionID)
	}

	nm.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"temp_node_id":  tempNodeID,
	}).Info("Processing E2 Setup Request")

	// Update node status
	node.Status = NodeStatusSetupInProgress

	// Send event
	nm.sendEvent(NodeEvent{
		Type:      NodeEventConnected,
		Node:      node,
		Timestamp: time.Now(),
	})

	// Extract node information from setup request
	globalNodeID := req.GlobalE2NodeID
	var realNodeID string
	var nodeType string

	// Determine node type and extract node ID
	if globalNodeID.GNBNodeID != nil {
		nodeType = string(E2NodeTypeGNB)
		realNodeID = fmt.Sprintf("gnb_%x", globalNodeID.GNBNodeID.PLMNIdentity)
		if globalNodeID.GNBNodeID.GNBCUUPId != nil {
			realNodeID += fmt.Sprintf("_cuup_%d", *globalNodeID.GNBNodeID.GNBCUUPId)
		}
		if globalNodeID.GNBNodeID.GNBCUCPId != nil {
			realNodeID += fmt.Sprintf("_cucp_%d", *globalNodeID.GNBNodeID.GNBCUCPId)
		}
		if globalNodeID.GNBNodeID.GNBDUId != nil {
			realNodeID += fmt.Sprintf("_du_%d", *globalNodeID.GNBNodeID.GNBDUId)
		}
	} else if globalNodeID.ENBNodeID != nil {
		nodeType = string(E2NodeTypeENB)
		realNodeID = fmt.Sprintf("enb_%x_%x", globalNodeID.ENBNodeID.PLMNIdentity, globalNodeID.ENBNodeID.ENBId)
	} else if globalNodeID.NGENBNodeID != nil {
		nodeType = string(E2NodeTypeNGENB)
		realNodeID = fmt.Sprintf("ng-enb_%x", globalNodeID.NGENBNodeID.PLMNIdentity)
	} else if globalNodeID.ENGNBNodeID != nil {
		nodeType = string(E2NodeTypeENGNB)
		realNodeID = fmt.Sprintf("en-gnb_%x_%x", globalNodeID.ENGNBNodeID.PLMNIdentity, globalNodeID.ENGNBNodeID.GNBId)
	} else {
		return fmt.Errorf("invalid global E2 node ID in setup request")
	}

	// Check if node with this ID already exists
	if existingNode, exists := nm.nodes[realNodeID]; exists && existingNode.ConnectionID != connectionID {
		nm.logger.WithFields(logrus.Fields{
			"existing_node_id":  realNodeID,
			"existing_conn_id":  existingNode.ConnectionID,
			"new_conn_id":       connectionID,
		}).Warn("Node ID conflict detected")
		
		return fmt.Errorf("node %s already exists with different connection", realNodeID)
	}

	// Update node with real information
	node.NodeID = realNodeID
	node.NodeType = nodeType
	node.GlobalE2NodeID = globalNodeID
	node.RANFunctions = req.RANFunctions
	node.ComponentConfigs = req.E2NodeComponentConfigAdd

	// Remove from temporary ID and add with real ID
	delete(nm.nodes, tempNodeID)
	nm.nodes[realNodeID] = node

	// Validate RAN functions
	acceptedFunctions, rejectedFunctions := nm.validateRANFunctions(req.RANFunctions)

	// TODO: Create setup response - currently unused
	_ = &E2SetupResponse{
		TransactionID: req.TransactionID,
		GlobalRICID: GlobalRICID{
			PLMNIdentity: []byte{0x02, 0xF8, 0x39}, // Example PLMN ID
			RICIdentity:  []byte{0x01, 0x02, 0x03, 0x04}, // Example RIC ID
		},
		RANFunctionsAccepted: acceptedFunctions,
		RANFunctionsRejected: rejectedFunctions,
	}

	// Update node status
	if len(rejectedFunctions) == 0 {
		node.Status = NodeStatusOperational
		node.ConfigVersion = "1.0"

		nm.logger.WithFields(logrus.Fields{
			"node_id":     realNodeID,
			"node_type":   nodeType,
			"ran_funcs":   len(acceptedFunctions),
		}).Info("E2 Setup completed successfully")

		// Send success event
		nm.sendEvent(NodeEvent{
			Type:      NodeEventSetupComplete,
			Node:      node,
			Timestamp: time.Now(),
		})
	} else {
		node.Status = NodeStatusSetupFailed

		nm.logger.WithFields(logrus.Fields{
			"node_id":         realNodeID,
			"accepted_funcs":  len(acceptedFunctions),
			"rejected_funcs":  len(rejectedFunctions),
		}).Warn("E2 Setup completed with rejected functions")

		// Send partial success event
		nm.sendEvent(NodeEvent{
			Type:      NodeEventSetupFailed,
			Node:      node,
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"accepted_functions": len(acceptedFunctions),
				"rejected_functions": len(rejectedFunctions),
			},
		})
	}

	return nil
}

// validateRANFunctions validates RAN functions from setup request
func (nm *NodeManager) validateRANFunctions(functions []RANFunction) ([]RANFunctionAccepted, []RANFunctionRejected) {
	var accepted []RANFunctionAccepted
	var rejected []RANFunctionRejected

	for _, function := range functions {
		// Basic validation - in production, this would be more sophisticated
		if function.RANFunctionID >= 0 && len(function.RANFunctionDefinition) > 0 {
			accepted = append(accepted, RANFunctionAccepted{
				RANFunctionID:       function.RANFunctionID,
				RANFunctionRevision: function.RANFunctionRevision,
			})
		} else {
			causeEnum := asn1.Enumerated(function.RANFunctionID)
			rejected = append(rejected, RANFunctionRejected{
				RANFunctionID: function.RANFunctionID,
				Cause: Cause{
					RIC: &causeEnum, // Simplified cause
				},
			})
		}
	}

	return accepted, rejected
}

// UnregisterNode removes a node from management
func (nm *NodeManager) UnregisterNode(nodeID string) error {
	nm.nodesMutex.Lock()
	defer nm.nodesMutex.Unlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// Update node status
	node.Status = NodeStatusDisconnected

	// Remove from nodes map
	delete(nm.nodes, nodeID)

	// Update metrics
	nm.metrics.E2Metrics.RegisteredNodes.Dec()

	nm.logger.WithField("node_id", nodeID).Info("E2 node unregistered")

	// Send event
	nm.sendEvent(NodeEvent{
		Type:      NodeEventDisconnected,
		Node:      node,
		Timestamp: time.Now(),
	})

	return nil
}

// GetNode retrieves a node by ID
func (nm *NodeManager) GetNode(nodeID string) (*E2Node, error) {
	nm.nodesMutex.RLock()
	defer nm.nodesMutex.RUnlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	return node, nil
}

// GetAllNodes returns all managed nodes
func (nm *NodeManager) GetAllNodes() []*E2Node {
	nm.nodesMutex.RLock()
	defer nm.nodesMutex.RUnlock()

	nodes := make([]*E2Node, 0, len(nm.nodes))
	for _, node := range nm.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// UpdateNodeActivity updates the last activity timestamp for a node
func (nm *NodeManager) UpdateNodeActivity(nodeID string) {
	nm.nodesMutex.Lock()
	defer nm.nodesMutex.Unlock()

	if node, exists := nm.nodes[nodeID]; exists {
		node.LastActivity = time.Now()
		node.MessagesReceived++
	}
}

// AddEventHandler adds an event handler
func (nm *NodeManager) AddEventHandler(handler NodeEventHandler) {
	nm.eventHandlers = append(nm.eventHandlers, handler)
}

// sendEvent sends an event to all registered handlers
func (nm *NodeManager) sendEvent(event NodeEvent) {
	for _, handler := range nm.eventHandlers {
		go func(h NodeEventHandler) {
			switch event.Type {
			case NodeEventConnected:
				h.OnNodeConnected(event.Node)
			case NodeEventDisconnected:
				h.OnNodeDisconnected(event.Node)
			case NodeEventSetupComplete:
				h.OnNodeSetupCompleted(event.Node)
			case NodeEventSetupFailed:
				h.OnNodeSetupFailed(event.Node, "Setup failed")
			case NodeEventHeartbeatMissed:
				h.OnNodeError(event.Node, event.Error)
			}
		}(handler)
	}
}

// staleNodeMonitor monitors for stale nodes and marks them as disconnected
func (nm *NodeManager) staleNodeMonitor() {
	defer nm.wg.Done()

	ticker := time.NewTicker(nm.heartbeatInterval)
	defer ticker.Stop()

	nm.logger.Debug("Starting stale node monitor")

	for {
		select {
		case <-nm.ctx.Done():
			nm.logger.Debug("Stale node monitor stopping")
			return
		case <-ticker.C:
			nm.checkStaleNodes()
		}
	}
}

// checkStaleNodes checks for stale nodes and updates their status
func (nm *NodeManager) checkStaleNodes() {
	nm.nodesMutex.RLock()
	staleNodes := make([]*E2Node, 0)
	
	for _, node := range nm.nodes {
		if node.IsStale(nm.staleThreshold) && node.IsConnected() {
			staleNodes = append(staleNodes, node)
		}
	}
	nm.nodesMutex.RUnlock()

	for _, node := range staleNodes {
		nm.logger.WithFields(logrus.Fields{
			"node_id":      node.NodeID,
			"last_activity": node.LastActivity,
			"threshold":    nm.staleThreshold,
		}).Warn("Node detected as stale")

		// Update node status
		nm.nodesMutex.Lock()
		node.Status = NodeStatusFaulty
		nm.nodesMutex.Unlock()

		// Send event
		nm.sendEvent(NodeEvent{
			Type:      NodeEventDisconnected,
			Node:      node,
			Timestamp: time.Now(),
		})
	}
}

// statisticsCollector periodically collects and logs node statistics
func (nm *NodeManager) statisticsCollector() {
	defer nm.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.collectStatistics()
		}
	}
}

// collectStatistics collects and updates node statistics
func (nm *NodeManager) collectStatistics() {
	nm.nodesMutex.RLock()
	defer nm.nodesMutex.RUnlock()

	totalNodes := len(nm.nodes)
	connectedNodes := 0
	operationalNodes := 0

	for _, node := range nm.nodes {
		if node.IsConnected() {
			connectedNodes++
		}
		if node.Status == NodeStatusOperational {
			operationalNodes++
		}

		// Update per-node metrics
		if node.IsConnected() {
			uptime := node.GetUptime()
			nm.metrics.E2Metrics.NodeUptime.WithLabelValues(node.NodeID).Set(uptime.Seconds())
		}
	}

	// Update overall metrics
	nm.metrics.E2Metrics.RegisteredNodes.Set(float64(totalNodes))

	nm.logger.WithFields(logrus.Fields{
		"total_nodes":       totalNodes,
		"connected_nodes":   connectedNodes,
		"operational_nodes": operationalNodes,
	}).Debug("Node statistics collected")
}

// GetNodeByConnectionID finds a node by its connection ID
func (nm *NodeManager) GetNodeByConnectionID(connectionID string) (*E2Node, error) {
	nm.nodesMutex.RLock()
	defer nm.nodesMutex.RUnlock()

	for _, node := range nm.nodes {
		if node.ConnectionID == connectionID {
			return node, nil
		}
	}

	return nil, fmt.Errorf("no node found for connection %s", connectionID)
}