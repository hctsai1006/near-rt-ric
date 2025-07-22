package e2

import (
	"context"
	"encoding/asn1"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// E2Interface represents the main E2 interface implementation
// Complies with O-RAN.WG3.E2AP-v03.00 specification
type E2Interface struct {
	config  *config.E2Config
	logger  *logrus.Logger
	metrics *monitoring.MetricsCollector

	// Core components
	sctpServer        *SCTPServer
	nodeManager       *NodeManager
	subscriptionManager *SubscriptionManager
	workerPool        *WorkerPool
	codec             *ASN1Codec

	// Control and synchronization
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mutex    sync.RWMutex

	// Event handlers
	eventHandlers []E2InterfaceEventHandler
}

// E2InterfaceEventHandler defines interface for E2 interface events
type E2InterfaceEventHandler interface {
	OnE2InterfaceStarted()
	OnE2InterfaceStopped()
	OnNodeConnected(node *E2Node)
	OnNodeDisconnected(node *E2Node)
	OnSubscriptionCreated(subscription *RICSubscription)
	OnSubscriptionDeleted(subscription *RICSubscription)
	OnError(err error)
}

// NewE2Interface creates a new E2 interface instance
func NewE2Interface(cfg *config.E2Config, logger *logrus.Logger, metrics *monitoring.MetricsCollector) (*E2Interface, error) {
	ctx, cancel := context.WithCancel(context.Background())

	e2 := &E2Interface{
		config:  cfg,
		logger:  logger.WithField("component", "e2-interface").Logger,
		metrics: metrics,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize ASN.1 codec
	codecConfig := &config.ASN1Config{
		Strict:           true,
		ValidateOnDecode: true,
	}
	e2.codec = NewASN1Codec(codecConfig, logger)

	// Initialize SCTP server
	sctpConfig := &SCTPConfig{
		ListenAddress:     cfg.ListenAddress,
		Port:              cfg.Port,
		MaxConnections:    cfg.MaxConnections,
		ConnectionTimeout: cfg.ConnectionTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
		BufferSize:        cfg.BufferSize,
		SCTP:              cfg.SCTP,
		Streams:           cfg.SCTP.Streams,
		MaxAttempts:       cfg.SCTP.MaxAttempts,
		RTOInitial:        cfg.SCTP.InitialRTO,
		RTOMin:            cfg.SCTP.RTOMin,
		RTOMax:            cfg.SCTP.MaxRTO,
	}

	var err error
	e2.sctpServer, err = NewSCTPServer(sctpConfig, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create SCTP server: %w", err)
	}

	// Initialize node manager
	e2.nodeManager = NewNodeManager(cfg, logger, metrics, e2.codec)

	// Initialize subscription manager
	e2.subscriptionManager = NewSubscriptionManager(cfg, logger, metrics, e2.codec)

	// Initialize worker pool
	e2.workerPool = NewWorkerPool(cfg, logger, metrics)
	e2.workerPool.SetMessageHandler(e2)

	// Set up SCTP server event handlers
	e2.sctpServer.SetMessageHandler(e2.handleSCTPMessage)
	e2.sctpServer.SetConnectionHandler(e2.handleConnectionEvent)

	// Set up node manager event handlers
	e2.nodeManager.AddEventHandler(e2)

	// Set up subscription manager event handlers
	e2.subscriptionManager.AddEventHandler(e2)

	e2.logger.Info("E2 interface initialized successfully")
	return e2, nil
}

// Start starts the E2 interface
func (e2 *E2Interface) Start(ctx context.Context) error {
	e2.mutex.Lock()
	defer e2.mutex.Unlock()

	if e2.running {
		return fmt.Errorf("E2 interface is already running")
	}

	e2.logger.WithFields(logrus.Fields{
		"sctp_address": fmt.Sprintf("%s:%d", e2.config.ListenAddress, e2.config.Port),
		"max_nodes":    e2.config.MaxConnections,
	}).Info("Starting O-RAN E2 interface")

	// Start worker pool first
	if err := e2.workerPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Start node manager
	if err := e2.nodeManager.Start(ctx); err != nil {
		e2.workerPool.Stop(ctx)
		return fmt.Errorf("failed to start node manager: %w", err)
	}

	// Start subscription manager
	if err := e2.subscriptionManager.Start(ctx); err != nil {
		e2.nodeManager.Stop(ctx)
		e2.workerPool.Stop(ctx)
		return fmt.Errorf("failed to start subscription manager: %w", err)
	}

	// Start SCTP server last
	if err := e2.sctpServer.Start(ctx); err != nil {
		e2.subscriptionManager.Stop(ctx)
		e2.nodeManager.Stop(ctx)
		e2.workerPool.Stop(ctx)
		return fmt.Errorf("failed to start SCTP server: %w", err)
	}

	e2.running = true
	e2.logger.Info("O-RAN E2 interface started successfully")

	// Send event
	for _, handler := range e2.eventHandlers {
		go handler.OnE2InterfaceStarted()
	}

	return nil
}

// Stop stops the E2 interface gracefully
func (e2 *E2Interface) Stop(ctx context.Context) error {
	e2.mutex.Lock()
	defer e2.mutex.Unlock()

	if !e2.running {
		return nil
	}

	e2.logger.Info("Stopping O-RAN E2 interface")

	// Stop components in reverse order
	if err := e2.sctpServer.Stop(ctx); err != nil {
		e2.logger.WithError(err).Error("Error stopping SCTP server")
	}

	if err := e2.subscriptionManager.Stop(ctx); err != nil {
		e2.logger.WithError(err).Error("Error stopping subscription manager")
	}

	if err := e2.nodeManager.Stop(ctx); err != nil {
		e2.logger.WithError(err).Error("Error stopping node manager")
	}

	if err := e2.workerPool.Stop(ctx); err != nil {
		e2.logger.WithError(err).Error("Error stopping worker pool")
	}

	// Cancel context
	e2.cancel()

	e2.running = false
	e2.logger.Info("O-RAN E2 interface stopped successfully")

	// Send event
	for _, handler := range e2.eventHandlers {
		go handler.OnE2InterfaceStopped()
	}

	return nil
}

// MessageHandler interface implementation

// HandleE2SetupRequest handles E2 Setup Request messages
func (e2 *E2Interface) HandleE2SetupRequest(connectionID, nodeID string, req *E2SetupRequest) (*E2SetupResponse, error) {
	e2.logger.WithFields(logrus.Fields{
		"connection_id":  connectionID,
		"transaction_id": req.TransactionID,
		"ran_functions":  len(req.RANFunctions),
	}).Info("Processing E2 Setup Request")

	// Process setup request through node manager
	if err := e2.nodeManager.HandleE2SetupRequest(connectionID, req); err != nil {
		return nil, fmt.Errorf("node manager failed to handle setup request: %w", err)
	}

	// Register connection with node ID mapping in SCTP server
	node, err := e2.nodeManager.GetNodeByConnectionID(connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node after setup: %w", err)
	}

	if err := e2.sctpServer.RegisterNodeConnection(connectionID, node.NodeID); err != nil {
		return nil, fmt.Errorf("failed to register node connection: %w", err)
	}

	// Create setup response
	response := &E2SetupResponse{
		TransactionID: req.TransactionID,
		GlobalRICID: GlobalRICID{
			PLMNIdentity: []byte{0x02, 0xF8, 0x39},
			RICIdentity:  []byte{0x01, 0x02, 0x03, 0x04},
		},
	}

	// Add accepted/rejected RAN functions
	for _, function := range req.RANFunctions {
		if function.RANFunctionID >= 0 && len(function.RANFunctionDefinition) > 0 {
			response.RANFunctionsAccepted = append(response.RANFunctionsAccepted, RANFunctionAccepted{
				RANFunctionID:       function.RANFunctionID,
				RANFunctionRevision: function.RANFunctionRevision,
			})
		} else {
			ricCause := asn1.Enumerated(function.RANFunctionID)
			response.RANFunctionsRejected = append(response.RANFunctionsRejected, RANFunctionRejected{
				RANFunctionID: function.RANFunctionID,
				Cause: Cause{
					RIC: &ricCause,
				},
			})
		}
	}

	return response, nil
}

// HandleE2SetupResponse handles E2 Setup Response messages  
func (e2 *E2Interface) HandleE2SetupResponse(connectionID, nodeID string, resp *E2SetupResponse) error {
	return nil // E2 Setup Response typically sent by RIC, not received
}

// HandleE2SetupFailure handles E2 Setup Failure messages
func (e2 *E2Interface) HandleE2SetupFailure(connectionID, nodeID string, failure *E2SetupFailure) error {
	return nil // Handle setup failure - typically disconnect the node
}

// HandleRICSubscriptionRequest handles RIC Subscription Request messages
func (e2 *E2Interface) HandleRICSubscriptionRequest(connectionID, nodeID string, req *RICSubscriptionRequest) (*RICSubscriptionResponse, error) {
	subscription, err := e2.subscriptionManager.CreateSubscription(nodeID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	response := &RICSubscriptionResponse{
		RICRequestID:  req.RICRequestID,
		RANFunctionID: req.RANFunctionID,
	}

	// For now, admit all actions
	for _, action := range req.RICSubscriptionDetails.RICActions {
		response.RICActionAdmitted = append(response.RICActionAdmitted, RICActionAdmitted{
			RICActionID: action.RICActionID,
		})
	}

	e2.logger.WithField("subscription_id", subscription.SubscriptionID).Info("RIC Subscription Request processed")
	return response, nil
}

// HandleRICSubscriptionResponse handles RIC Subscription Response messages
func (e2 *E2Interface) HandleRICSubscriptionResponse(connectionID, nodeID string, resp *RICSubscriptionResponse) error {
	return e2.subscriptionManager.ProcessSubscriptionResponse(nodeID, resp)
}

// HandleRICSubscriptionFailure handles RIC Subscription Failure messages
func (e2 *E2Interface) HandleRICSubscriptionFailure(connectionID, nodeID string, failure *RICSubscriptionFailure) error {
	return e2.subscriptionManager.ProcessSubscriptionFailure(nodeID, failure)
}

// HandleRICSubscriptionDeleteRequest handles RIC Subscription Delete Request messages
func (e2 *E2Interface) HandleRICSubscriptionDeleteRequest(connectionID, nodeID string, req *RICSubscriptionDeleteRequest) (*RICSubscriptionDeleteResponse, error) {
	response := &RICSubscriptionDeleteResponse{
		RICRequestID:  req.RICRequestID,
		RANFunctionID: req.RANFunctionID,
	}
	return response, nil
}

// HandleRICSubscriptionDeleteResponse handles RIC Subscription Delete Response messages
func (e2 *E2Interface) HandleRICSubscriptionDeleteResponse(connectionID, nodeID string, resp *RICSubscriptionDeleteResponse) error {
	return e2.subscriptionManager.ProcessSubscriptionDeleteResponse(nodeID, resp)
}

// HandleRICIndication handles RIC Indication messages
func (e2 *E2Interface) HandleRICIndication(connectionID, nodeID string, indication *RICIndication) error {
	return e2.subscriptionManager.ProcessIndicationMessage(nodeID, indication)
}

// HandleRICControlRequest handles RIC Control Request messages
func (e2 *E2Interface) HandleRICControlRequest(connectionID, nodeID string, req *RICControlRequest) (*RICControlAck, error) {
	ack := &RICControlAck{
		RICRequestID:  req.RICRequestID,
		RANFunctionID: req.RANFunctionID,
	}
	if req.RICCallProcessID != nil {
		ack.RICCallProcessID = req.RICCallProcessID
	}
	return ack, nil
}

// HandleRICControlAck handles RIC Control Acknowledge messages
func (e2 *E2Interface) HandleRICControlAck(connectionID, nodeID string, ack *RICControlAck) error {
	return nil
}

// HandleRICControlFailure handles RIC Control Failure messages
func (e2 *E2Interface) HandleRICControlFailure(connectionID, nodeID string, failure *RICControlFailure) error {
	return nil
}

// NodeEventHandler interface implementation
func (e2 *E2Interface) OnNodeConnected(node *E2Node) {
	for _, handler := range e2.eventHandlers {
		go handler.OnNodeConnected(node)
	}
}

func (e2 *E2Interface) OnNodeDisconnected(node *E2Node) {
	for _, handler := range e2.eventHandlers {
		go handler.OnNodeDisconnected(node)
	}
}

func (e2 *E2Interface) OnNodeSetupCompleted(node *E2Node) {}
func (e2 *E2Interface) OnNodeSetupFailed(node *E2Node, reason string) {}
func (e2 *E2Interface) OnNodeError(node *E2Node, err error) {
	for _, handler := range e2.eventHandlers {
		go handler.OnError(err)
	}
}

// SubscriptionEventHandler interface implementation
func (e2 *E2Interface) OnSubscriptionCreated(subscription *RICSubscription) {
	for _, handler := range e2.eventHandlers {
		go handler.OnSubscriptionCreated(subscription)
	}
}

func (e2 *E2Interface) OnSubscriptionUpdated(subscription *RICSubscription) {}

func (e2 *E2Interface) OnSubscriptionDeleted(subscription *RICSubscription) {
	for _, handler := range e2.eventHandlers {
		go handler.OnSubscriptionDeleted(subscription)
	}
}

func (e2 *E2Interface) OnSubscriptionExpired(subscription *RICSubscription) {}
func (e2 *E2Interface) OnSubscriptionError(subscription *RICSubscription, err error) {}

// handleSCTPMessage handles incoming SCTP messages
func (e2 *E2Interface) handleSCTPMessage(connectionID, nodeID string, data []byte) {
	start := time.Now()

	// Determine message type
	messageType, err := e2.codec.GetMessageType(data)
	if err != nil {
		e2.logger.WithError(err).Error("Failed to determine message type")
		return
	}

	// Create E2 message for processing
	msg := &E2Message{
		MessageID:     uuid.New().String(),
		ConnectionID:  connectionID,
		NodeID:        nodeID,
		Data:          data,
		MessageType:   messageType,
		Timestamp:     time.Now(),
		Status:        "received",
		Size:          len(data),
	}

	// Update node activity
	if nodeID != "" {
		e2.nodeManager.UpdateNodeActivity(nodeID)
	}

	// Submit to worker pool for processing
	if err := e2.workerPool.SubmitMessage(msg); err != nil {
		e2.logger.WithError(err).Error("Failed to submit message to worker pool")
		e2.metrics.E2Metrics.MessagesTotal.WithLabelValues(nodeID, messageType.String(), "inbound", "failed").Inc()
		return
	}

	// Record metrics
	latency := time.Since(start)
	e2.metrics.RecordE2Message(nodeID, messageType.String(), "inbound", "success", len(data), latency)
}

// handleConnectionEvent handles SCTP connection events
func (e2 *E2Interface) handleConnectionEvent(event *ConnectionEvent) {
	switch event.Type {
	case ConnectionEstablished:
		node, err := e2.nodeManager.RegisterNode(event.ConnectionID, event.RemoteAddr)
		if err != nil {
			e2.logger.WithError(err).Error("Failed to register new node")
			return
		}
		e2.logger.WithField("node_id", node.NodeID).Info("New E2 node connected")

	case ConnectionClosed:
		if event.NodeID != "" {
			e2.subscriptionManager.CleanupNodeSubscriptions(event.NodeID)
			e2.nodeManager.UnregisterNode(event.NodeID)
			e2.logger.WithField("node_id", event.NodeID).Info("E2 node disconnected")
		}
	}
}

// Public interface methods

// IsRunning returns whether the E2 interface is running
func (e2 *E2Interface) IsRunning() bool {
	e2.mutex.RLock()
	defer e2.mutex.RUnlock()
	return e2.running
}

// AddEventHandler adds an event handler
func (e2 *E2Interface) AddEventHandler(handler E2InterfaceEventHandler) {
	e2.eventHandlers = append(e2.eventHandlers, handler)
}

// GetStats returns E2 interface statistics
func (e2 *E2Interface) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"sctp_stats":        e2.sctpServer.GetStats(),
		"worker_pool_stats": e2.workerPool.GetStats(),
		"total_nodes":       len(e2.nodeManager.GetAllNodes()),
		"total_subscriptions": len(e2.subscriptionManager.GetAllSubscriptions()),
		"running":           e2.IsRunning(),
	}
}

// HealthCheck performs a health check of the E2 interface
func (e2 *E2Interface) HealthCheck() error {
	if !e2.IsRunning() {
		return fmt.Errorf("E2 interface is not running")
	}
	return e2.sctpServer.HealthCheck()
}

// GetNodes returns all connected E2 nodes
func (e2 *E2Interface) GetNodes() []*E2Node {
	return e2.nodeManager.GetAllNodes()
}

// GetNode returns a specific E2 node by ID
func (e2 *E2Interface) GetNode(nodeID string) (*E2Node, error) {
	return e2.nodeManager.GetNode(nodeID)
}

// GetSubscriptions returns all active subscriptions
func (e2 *E2Interface) GetSubscriptions() []*RICSubscription {
	return e2.subscriptionManager.GetAllSubscriptions()
}

// GetSubscriptionsByNode returns subscriptions for a specific node
func (e2 *E2Interface) GetSubscriptionsByNode(nodeID string) []*RICSubscription {
	return e2.subscriptionManager.GetSubscriptionsByNode(nodeID)
}