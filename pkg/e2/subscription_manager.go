package e2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
	"github.com/sirupsen/logrus"
)

// SubscriptionManager manages RIC subscriptions according to O-RAN E2AP specification
type SubscriptionManager struct {
	config  *config.E2Config
	logger  *logrus.Logger
	metrics *monitoring.MetricsCollector
	codec   *ASN1Codec

	// Subscription storage
	subscriptions      map[string]*models.RICSubscription
	subscriptionsMutex sync.RWMutex
	
	// Indexing for fast lookups
	subscriptionsByNode    map[string][]*models.RICSubscription
	subscriptionsByRequest map[string]*models.RICSubscription
	
	// Event handling
	eventHandlers []SubscriptionEventHandler

	// Background tasks
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Configuration
	defaultTimeout time.Duration
	cleanupInterval time.Duration
}

// SubscriptionEventHandler defines interface for handling subscription events
type SubscriptionEventHandler interface {
	OnSubscriptionCreated(subscription *models.RICSubscription)
	OnSubscriptionUpdated(subscription *models.RICSubscription)
	OnSubscriptionDeleted(subscription *models.RICSubscription)
	OnSubscriptionExpired(subscription *models.RICSubscription)
	OnSubscriptionError(subscription *models.RICSubscription, err error)
}

// SubscriptionEvent represents a subscription lifecycle event
type SubscriptionEvent struct {
	Type         SubscriptionEventType
	Subscription *models.RICSubscription
	Timestamp    time.Time
	Error        error
	Details      map[string]interface{}
}

// SubscriptionEventType represents the type of subscription event
type SubscriptionEventType int

const (
	SubscriptionEventCreated SubscriptionEventType = iota
	SubscriptionEventUpdated
	SubscriptionEventDeleted
	SubscriptionEventExpired
	SubscriptionEventError
	SubscriptionEventIndicationReceived
)

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(config *config.E2Config, logger *logrus.Logger, metrics *monitoring.MetricsCollector, codec *ASN1Codec) *SubscriptionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &SubscriptionManager{
		config:                 config,
				logger:                 logger.WithField("component", "subscription-manager"),
		metrics:               metrics,
		codec:                 codec,
		subscriptions:         make(map[string]*models.RICSubscription),
		subscriptionsByNode:   make(map[string][]*models.RICSubscription),
		subscriptionsByRequest: make(map[string]*models.RICSubscription),
		ctx:                   ctx,
		cancel:                cancel,
		defaultTimeout:        30 * time.Second,
		cleanupInterval:       60 * time.Second,
	}
}

// CreateSubscription creates a new RIC subscription
func (sm *SubscriptionManager) CreateSubscription(nodeID string, req *models.RICSubscriptionRequest) (*models.RICSubscription, error) {
	sm.subscriptionsMutex.Lock()
	defer sm.subscriptionsMutex.Unlock()

	// Generate unique subscription ID
	subscriptionID := uuid.New().String()

	// Create request key for tracking
	requestKey := sm.getRequestKey(nodeID, *req.RICrequestID)

	// Check if subscription already exists for this request
	if _, exists := sm.subscriptionsByRequest[requestKey]; exists {
		return nil, fmt.Errorf("subscription already exists for node %s, request %s", nodeID, requestKey)
	}

	// Validate subscription request
	if err := sm.validateSubscriptionRequest(req); err != nil {
		return nil, fmt.Errorf("invalid subscription request: %w", err)
	}

	// Create subscription
	subscription := &models.RICSubscription{
		RequestID:            *req.RICrequestID,
		SubscriptionID:       subscriptionID,
		NodeID:              nodeID,
		RANFunctionID:       req.RANfunctionID,
		SubscriptionDetails: *req.RICsubscriptionDetails,
		Status:              models.SubscriptionStatusPending,
		CreatedAt:           time.Now(),
		LastIndication:      time.Time{},
		Actions:             req.RICsubscriptionDetails.RICactions,
		AdmittedActions:     make([]int64, 0),
		RejectedActions:     make([]models.RICActionNotAdmitted, 0),
	}

	// Set expiration if configured
	if sm.config.SubscriptionTimeout > 0 {
		expiresAt := time.Now().Add(time.Duration(sm.config.SubscriptionTimeout) * time.Second)
		subscription.ExpiresAt = &expiresAt
	}

	// Store subscription
	sm.subscriptions[subscriptionID] = subscription
	sm.subscriptionsByRequest[requestKey] = subscription

	// Add to node index
	if sm.subscriptionsByNode[nodeID] == nil {
		sm.subscriptionsByNode[nodeID] = make([]*models.RICSubscription, 0)
	}
	sm.subscriptionsByNode[nodeID] = append(sm.subscriptionsByNode[nodeID], subscription)

	// Update metrics
	sm.metrics.E2Metrics.ActiveSubscriptions.Inc()

	sm.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
		"node_id":        nodeID,
		"ran_function_id": req.RANfunctionID,
		"request_id":     fmt.Sprintf("%d-%d", req.RICrequestID.RICrequestorID, req.RICrequestID.RICInstanceID),
	}).Info("RIC subscription created")

	// Send event
	sm.sendEvent(SubscriptionEvent{
		Type:         SubscriptionEventCreated,
		Subscription: subscription,
		Timestamp:    time.Now(),
	})

	return subscription, nil
}

// ProcessSubscriptionResponse processes a subscription response from E2 node
func (sm *SubscriptionManager) ProcessSubscriptionResponse(nodeID string, resp *models.RICSubscriptionResponse) error {
	sm.subscriptionsMutex.Lock()
	defer sm.subscriptionsMutex.Unlock()

	// Find subscription by request ID
	requestKey := sm.getRequestKey(nodeID, resp.RICRequestID)
	subscription, exists := sm.subscriptionsByRequest[requestKey]
	if !exists {
		return fmt.Errorf("no subscription found for node %s, request %s", nodeID, requestKey)
	}

	// Update subscription with response
	subscription.Status = models.SubscriptionStatusActive
	subscription.AdmittedActions = make([]int64, len(resp.RICActionAdmitted))
	for i, action := range resp.RICActionAdmitted {
		subscription.AdmittedActions[i] = action.RICActionID
	}

	subscription.RejectedActions = resp.RICActionNotAdmitted

	sm.logger.WithFields(logrus.Fields{
		"subscription_id":   subscription.SubscriptionID,
		"node_id":          nodeID,
		"admitted_actions": len(resp.RICActionAdmitted),
		"rejected_actions": len(resp.RICActionNotAdmitted),
	}).Info("RIC subscription response processed")

	// Send event
	sm.sendEvent(SubscriptionEvent{
		Type:         SubscriptionEventUpdated,
		Subscription: subscription,
		Timestamp:    time.Now(),
		Details: map[string]interface{}{
			"admitted_actions": len(resp.RICActionAdmitted),
			"rejected_actions": len(resp.RICActionNotAdmitted),
		},
	})

	return nil
}

// ProcessSubscriptionFailure processes a subscription failure from E2 node
func (sm *SubscriptionManager) ProcessSubscriptionFailure(nodeID string, failure *models.RICSubscriptionFailure) error {
	sm.subscriptionsMutex.Lock()
	defer sm.subscriptionsMutex.Unlock()

	// Find subscription by request ID
	requestKey := sm.getRequestKey(nodeID, failure.RICRequestID)
	subscription, exists := sm.subscriptionsByRequest[requestKey]
	if !exists {
		return fmt.Errorf("no subscription found for node %s, request %s", nodeID, requestKey)
	}

	// Update subscription status
	subscription.Status = models.SubscriptionStatusFailed
	subscription.RejectedActions = failure.RICActionNotAdmitted
	subscription.ErrorCount++

	// Update metrics
	sm.metrics.E2Metrics.SubscriptionErrors.Inc()

	sm.logger.WithFields(logrus.Fields{
		"subscription_id":   subscription.SubscriptionID,
		"node_id":          nodeID,
		"rejected_actions": len(failure.RICActionNotAdmitted),
	}).Warn("RIC subscription failed")

	// Send event
	sm.sendEvent(SubscriptionEvent{
		Type:         SubscriptionEventError,
		Subscription: subscription,
		Timestamp:    time.Now(),
		Details: map[string]interface{}{
			"rejected_actions": len(failure.RICActionNotAdmitted),
		},
	})

	return nil
}

// ProcessSubscriptionDeleteResponse processes a subscription delete response
func (sm *SubscriptionManager) ProcessSubscriptionDeleteResponse(nodeID string, resp *models.RICSubscriptionDeleteResponse) error {
	requestKey := sm.getRequestKey(nodeID, resp.RICRequestID)
	
	sm.subscriptionsMutex.RLock()
	subscription, exists := sm.subscriptionsByRequest[requestKey]
	sm.subscriptionsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("no subscription found for delete response: node %s, request %s", nodeID, requestKey)
	}

	// Delete the subscription
	return sm.DeleteSubscription(subscription.SubscriptionID)
}

// ProcessIndicationMessage processes a RIC indication message
func (sm *SubscriptionManager) ProcessIndicationMessage(nodeID string, indication *models.RICIndication) error {
	sm.subscriptionsMutex.Lock()
	defer sm.subscriptionsMutex.Unlock()

	// Find subscription by request ID
	requestKey := sm.getRequestKey(nodeID, indication.RICRequestID)
	subscription, exists := sm.subscriptionsByRequest[requestKey]
	if !exists {
		return fmt.Errorf("no subscription found for indication: node %s, request %s", nodeID, requestKey)
	}

	// Update subscription statistics
	subscription.LastIndication = time.Now()
	subscription.IndicationsReceived++

	sm.logger.WithFields(logrus.Fields{
		"subscription_id":     subscription.SubscriptionID,
		"node_id":            nodeID,
		"ran_function_id":    indication.RANFunctionID,
		"action_id":          indication.RICActionID,
		"indication_sn":      indication.RICIndicationSN,
		"indication_type":    indication.RICIndicationType,
		"message_size":       len(indication.RICIndicationMessage),
	}).Debug("RIC indication processed")

	// Send event
	sm.sendEvent(SubscriptionEvent{
		Type:         SubscriptionEventIndicationReceived,
		Subscription: subscription,
		Timestamp:    time.Now(),
		Details: map[string]interface{}{
			"indication_size": len(indication.RICIndicationMessage),
			"action_id":      indication.RICActionID,
		},
	})

	return nil
}

// GetSubscription retrieves a subscription by ID
func (sm *SubscriptionManager) GetSubscription(subscriptionID string) (*models.RICSubscription, error) {
	sm.subscriptionsMutex.RLock()
	defer sm.subscriptionsMutex.RUnlock()

	subscription, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}

	return subscription, nil
}

// GetSubscriptionsByNode retrieves all subscriptions for a node
func (sm *SubscriptionManager) GetSubscriptionsByNode(nodeID string) []*models.RICSubscription {
	sm.subscriptionsMutex.RLock()
	defer sm.subscriptionsMutex.RUnlock()

	subscriptions := sm.subscriptionsByNode[nodeID]
	if subscriptions == nil {
		return make([]*models.RICSubscription, 0)
	}

	// Return a copy to prevent external modification
	result := make([]*models.RICSubscription, len(subscriptions))
	copy(result, subscriptions)
	return result
}

// GetAllSubscriptions returns all active subscriptions
func (sm *SubscriptionManager) GetAllSubscriptions() []*models.RICSubscription {
	sm.subscriptionsMutex.RLock()
	defer sm.subscriptionsMutex.RUnlock()

	subscriptions := make([]*models.RICSubscription, 0, len(sm.subscriptions))
	for _, subscription := range sm.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions
}

// validateSubscriptionRequest validates a subscription request
func (sm *SubscriptionManager) validateSubscriptionRequest(req *models.RICSubscriptionRequest) error {
	if req.RICrequestID.RICRequestorID < 0 {
		return fmt.Errorf("invalid RIC requestor ID: %d", req.RICrequestID.RICRequestorID)
	}

	if req.RICrequestID.RICInstanceID < 0 {
		return fmt.Errorf("invalid RIC instance ID: %d", req.RICrequestID.RICInstanceID)
	}

	if req.RANfunctionID < 0 {
		return fmt.Errorf("invalid RAN function ID: %d", req.RANfunctionID)
	}

	if len(req.RICsubscriptionDetails.RICEventTriggerDefinition) == 0 {
		return fmt.Errorf("RIC event trigger definition is required")
	}

	if len(req.RICsubscriptionDetails.RICActions) == 0 {
		return fmt.Errorf("at least one RIC action is required")
	}

	// Validate each action
	for i, action := range req.RICsubscriptionDetails.RICActions {
		if action.RICActionID < 0 {
			return fmt.Errorf("invalid RIC action ID at index %d: %d", i, action.RICActionID)
		}
	}

	return nil
}

// getRequestKey generates a unique key for a subscription request
func (sm *SubscriptionManager) getRequestKey(nodeID string, requestID models.RICrequestID) string {
	return fmt.Sprintf("%s_%d_%d", nodeID, requestID.RICRequestorID, requestID.RICInstanceID)
}

// checkExpiredSubscriptions checks for and handles expired subscriptions
func (sm *SubscriptionManager) checkExpiredSubscriptions() {
	sm.subscriptionsMutex.RLock()
	expiredSubscriptions := make([]*models.RICSubscription, 0)
	
	for _, subscription := range sm.subscriptions {
		if subscription.IsExpired() {
			expiredSubscriptions = append(expiredSubscriptions, subscription)
		}
	}
	sm.subscriptionsMutex.RUnlock()

	for _, subscription := range expiredSubscriptions {
		sm.logger.WithFields(logrus.Fields{
			"subscription_id": subscription.SubscriptionID,
			"node_id":        subscription.NodeID,
			"expires_at":     subscription.ExpiresAt,
		}).Info("Subscription expired")

		// Update status
		sm.subscriptionsMutex.Lock()
		subscription.Status = models.SubscriptionStatusExpired
		sm.subscriptionsMutex.Unlock()

		// Send event
		sm.sendEvent(SubscriptionEvent{
			Type:         SubscriptionEventExpired,
			Subscription: subscription,
			Timestamp:    time.Now(),
		})

		// Clean up expired subscription
		sm.DeleteSubscription(subscription.SubscriptionID)
	}
}

// collectStatistics collects and updates subscription statistics
func (sm *SubscriptionManager) collectStatistics() {
	sm.subscriptionsMutex.RLock()
	defer sm.subscriptionsMutex.RUnlock()

	totalSubscriptions := len(sm.subscriptions)
	activeSubscriptions := 0
	pendingSubscriptions := 0
	failedSubscriptions := 0

	for _, subscription := range sm.subscriptions {
		switch subscription.Status {
		case models.SubscriptionStatusActive:
			activeSubscriptions++
		case models.SubscriptionStatusPending:
			pendingSubscriptions++
		case models.SubscriptionStatusFailed:
			failedSubscriptions++
		}
	}

	// Update metrics
	sm.metrics.E2Metrics.ActiveSubscriptions.Set(float64(activeSubscriptions))

	sm.logger.WithFields(logrus.Fields{
		"total_subscriptions":   totalSubscriptions,
		"active_subscriptions":  activeSubscriptions,
		"pending_subscriptions": pendingSubscriptions,
		"failed_subscriptions":  failedSubscriptions,
	}).Debug("Subscription statistics collected")
}
