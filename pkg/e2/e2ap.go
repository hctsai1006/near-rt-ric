package e2

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/node_manager"
)

// E2APProcessor handles E2AP procedures according to O-RAN specifications
type E2APProcessor struct {
	e2Interface  *E2Interface
	nodeManager  *node_manager.E2NodeManager
	logger       *logrus.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	transactions map[int64]*Transaction
	nextTransID  int64
}

// Transaction represents an ongoing E2AP transaction
type Transaction struct {
	ID        int32
	Type      TransactionType
	NodeID    string
	StartTime time.Time
	Timeout   time.Duration
	Context   context.Context
	Cancel    context.CancelFunc
	Response  chan *models.E2APMessage
}

// TransactionType represents the type of E2AP transaction
type TransactionType int

const (
	TransactionTypeE2Setup TransactionType = iota
	TransactionTypeSubscription
	TransactionTypeControl
	TransactionTypeServiceUpdate
)

// NewE2APProcessor creates a new O-RAN compliant E2APProcessor
func NewE2APProcessor(e2Interface *E2Interface, nodeManager *node_manager.E2NodeManager) *E2APProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	processor := &E2APProcessor{
		e2Interface:  e2Interface,
		nodeManager:  nodeManager,
		logger:       logrus.WithField("component", "e2ap-processor"),
		ctx:          ctx,
		cancel:       cancel,
		transactions: make(map[int64]*Transaction),
		nextTransID:  1,
	}
	
	// Start transaction cleanup routine
	go processor.transactionCleanupRoutine()
	
	return processor
}

// ProcessE2SetupRequest processes an E2 setup request according to O-RAN E2AP specification
func (p *E2APProcessor) ProcessE2SetupRequest(nodeID string, request *models.E2SetupRequest) error {
	start := time.Now()
	p.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
	}).Info("Processing E2 Setup Request")

	// Register the E2 node
	node := &models.E2Node{
		NodeID:           nodeID,
		NodeType:         string(models.E2NodeTypeGNB),
		GlobalE2NodeID:   request.GlobalE2NodeID,
		RemoteAddress:    "", // Will be set by SCTP manager
		Status:           models.NodeStatusSetupInProgress,
		LastHeartbeat:    time.Now(),
	}

	// Add supported functions
	node.RANFunctions = request.RANFunctions

	p.nodeManager.AddNode(*node)

	// Send E2 Setup Response
	response := &models.E2SetupResponse{
		TransactionID: request.TransactionID,
		GlobalRICID: models.GlobalRICID{
			PLMNIdentity: []byte{0x00, 0xF1, 0x10},
			RICIdentity:  []byte{0x00, 0x00, 0x00, 0x01}, // RIC Instance ID
		},
		RANFunctionsAccepted: []models.RANFunctionAccepted{
			{
				RANFunctionID:       1,
				RANFunctionRevision: 1,
			},
		},
	}

	// Send response via E2 interface
	if _, err := p.e2Interface.SendE2SetupRequest(nodeID, response); err != nil {
		p.logger.WithError(err).Error("Failed to send E2 Setup Response")
		return err
	}

	// Update node status
	node.Status = models.NodeStatusOperational
	node.ConnectedAt = time.Now()
	p.nodeManager.UpdateNode(nodeID, *node)

	p.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"duration": time.Since(start),
		"functions": len(request.RANFunctions),
	}).Info("E2 Setup Request processed successfully")

	return nil
}

// ProcessSubscriptionRequest processes a RIC subscription request
func (p *E2APProcessor) ProcessSubscriptionRequest(nodeID string, request *models.RICSubscriptionRequest) error {
	start := time.Now()
	p.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
	}).Info("Processing RIC Subscription Request")

	// Store subscription (would be in a proper subscription manager)
	p.logger.WithFields(logrus.Fields{
		"requestor_id": request.RICrequestID.RICrequestorID,
		"instance_id":  request.RICrequestID.RICinstanceID,
		"function_id":  request.RANfunctionID,
	}).Info("RIC Subscription stored")

	// Create and send subscription response
	response := &models.RICSubscriptionResponse{}

	// Send response via E2 interface
	if _, err := p.e2Interface.CreateSubscription(nodeID, response); err != nil {
		p.logger.WithError(err).Error("Failed to send RIC Subscription Response")
		return err
	}

	p.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"duration": time.Since(start),
	}).Info("RIC Subscription Request processed successfully")

	return nil
}

// ProcessIndication processes a RIC indication message
func (p *E2APProcessor) ProcessIndication(nodeID string, indication *models.RICIndication) error {
	start := time.Now()
	p.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
	}).Info("Processing RIC Indication")

	// Process indication content (would forward to xApps)
	p.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"duration": time.Since(start),
	}).Info("RIC Indication processed successfully")

	return nil
}

// Helper methods

func (p *E2APProcessor) sendE2SetupFailure(nodeID string, causeType models.CauseType, causeValue int32) error {
	// TODO: Implement E2 Setup Failure encoding and sending
	p.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"cause_type": causeType,
		"cause_value": causeValue,
	}).Error("Sending E2 Setup Failure")
	return nil
}

func (p *E2APProcessor) sendSubscriptionFailure(nodeID string, causeType models.CauseType, causeValue int32) error {
	// TODO: Implement RIC Subscription Failure encoding and sending
	p.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"cause_type": causeType,
		"cause_value": causeValue,
	}).Error("Sending RIC Subscription Failure")
	return nil
}

func (p *E2APProcessor) getNextTransactionID() int64 {
	id := p.nextTransID
	p.nextTransID++
	if p.nextTransID > 1000000 {
		p.nextTransID = 1 // Reset to avoid overflow
	}
	return id
}

func (p *E2APProcessor) transactionCleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			for id, txn := range p.transactions {
				if now.Sub(txn.StartTime) > txn.Timeout {
					p.logger.WithField("transaction_id", id).Warn("Transaction timeout, cleaning up")
					txn.Cancel()
					delete(p.transactions, id)
				}
			}
		}
	}
}

// Cleanup stops the E2AP processor and cleans up resources
func (p *E2APProcessor) Cleanup() {
	p.cancel()
	p.logger.Info("E2AP Processor cleanup completed")
}
