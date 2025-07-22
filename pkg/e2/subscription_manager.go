package e2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// SubscriptionManager manages RIC subscriptions according to O-RAN E2AP specification
type SubscriptionManager struct {
	config  *config.E2Config
	logger  *logrus.Logger
	metrics *monitoring.MetricsCollector
	codec   *ASN1Codec

	// Subscription storage
	subscriptions      map[string]*RICSubscription
	subscriptionsMutex sync.RWMutex
	
	// Indexing for fast lookups
	subscriptionsByNode    map[string][]*RICSubscription
	subscriptionsByRequest map[string]*RICSubscription
	
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
	OnSubscriptionCreated(subscription *RICSubscription)
	OnSubscriptionUpdated(subscription *RICSubscription)
	OnSubscriptionDeleted(subscription *RICSubscription)
	OnSubscriptionExpired(subscription *RICSubscription)
	OnSubscriptionError(subscription *RICSubscription, err error)
}

// SubscriptionEvent represents a subscription lifecycle event
type SubscriptionEvent struct {
	Type         SubscriptionEventType
	Subscription *RICSubscription
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
		logger:                 logger.WithField("component", "subscription-manager").Logger,
		metrics:               metrics,
		codec:                 codec,
		subscriptions:         make(map[string]*RICSubscription),
		subscriptionsByNode:   make(map[string][]*RICSubscription),
		subscriptionsByRequest: make(map[string]*RICSubscription),
		ctx:                   ctx,
		cancel:                cancel,
		defaultTimeout:        30 * time.Second,
		cleanupInterval:       60 * time.Second,
	}
}

// Start starts the subscription manager
func (sm *SubscriptionManager) Start(ctx context.Context) error {
	sm.logger.Info("Starting E2 Subscription Manager")

	// Start background workers
	sm.wg.Add(2)
	go sm.expirationMonitor()
	go sm.statisticsCollector()

	sm.logger.Info("E2 Subscription Manager started successfully")
	return nil
}

// Stop stops the subscription manager
func (sm *SubscriptionManager) Stop(ctx context.Context) error {
	sm.logger.Info("Stopping E2 Subscription Manager")

	// Cancel context to stop background workers
	sm.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		sm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		sm.logger.Info("E2 Subscription Manager stopped successfully")
	case <-ctx.Done():
		sm.logger.Warn("E2 Subscription Manager shutdown timeout")
	}

	return nil
}

// CreateSubscription creates a new RIC subscription
func (sm *SubscriptionManager) CreateSubscription(nodeID string, req *RICSubscriptionRequest) (*RICSubscription, error) {
	sm.subscriptionsMutex.Lock()
	defer sm.subscriptionsMutex.Unlock()

	// Generate unique subscription ID
	subscriptionID := uuid.New().String()

	// Create request key for tracking
	requestKey := sm.getRequestKey(nodeID, req.RICRequestID)

	// Check if subscription already exists for this request
	if _, exists := sm.subscriptionsByRequest[requestKey]; exists {
		return nil, fmt.Errorf("subscription already exists for node %s, request %s", nodeID, requestKey)
	}

	// Validate subscription request
	if err := sm.validateSubscriptionRequest(req); err != nil {
		return nil, fmt.Errorf("invalid subscription request: %w", err)
	}

	// Create subscription
	subscription := &RICSubscription{
		RequestID:            req.RICRequestID,
		SubscriptionID:       subscriptionID,
		NodeID:              nodeID,
		RANFunctionID:       req.RANFunctionID,
		SubscriptionDetails: req.RICSubscriptionDetails,
		Status:              SubscriptionStatusPending,
		CreatedAt:           time.Now(),
		LastIndication:      time.Time{},
		Actions:             req.RICSubscriptionDetails.RICActions,
		AdmittedActions:     make([]int64, 0),
		RejectedActions:     make([]RICActionNotAdmitted, 0),
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
		sm.subscriptionsByNode[nodeID] = make([]*RICSubscription, 0)
	}
	sm.subscriptionsByNode[nodeID] = append(sm.subscriptionsByNode[nodeID], subscription)

	// Update metrics
	sm.metrics.E2Metrics.ActiveSubscriptions.Inc()

	sm.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
		"node_id":        nodeID,
		"ran_function_id": req.RANFunctionID,
		"request_id":     fmt.Sprintf("%d-%d", req.RICRequestID.RICRequestorID, req.RICRequestID.RICInstanceID),
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
func (sm *SubscriptionManager) ProcessSubscriptionResponse(nodeID string, resp *RICSubscriptionResponse) error {
	sm.subscriptionsMutex.Lock()
	defer sm.subscriptionsMutex.Unlock()

	// Find subscription by request ID
	requestKey := sm.getRequestKey(nodeID, resp.RICRequestID)
	subscription, exists := sm.subscriptionsByRequest[requestKey]
	if !exists {
		return fmt.Errorf("no subscription found for node %s, request %s", nodeID, requestKey)
	}

	// Update subscription with response
	subscription.Status = SubscriptionStatusActive
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
func (sm *SubscriptionManager) ProcessSubscriptionFailure(nodeID string, failure *RICSubscriptionFailure) error {
	sm.subscriptionsMutex.Lock()
	defer sm.subscriptionsMutex.Unlock()

	// Find subscription by request ID
	requestKey := sm.getRequestKey(nodeID, failure.RICRequestID)
	subscription, exists := sm.subscriptionsByRequest[requestKey]
	if !exists {
		return fmt.Errorf("no subscription found for node %s, request %s", nodeID, requestKey)
	}

	// Update subscription status
	subscription.Status = SubscriptionStatusFailed
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

// DeleteSubscription deletes a RIC subscription
func (sm *SubscriptionManager) DeleteSubscription(subscriptionID string) error {
	sm.subscriptionsMutex.Lock()
	defer sm.subscriptionsMutex.Unlock()

	subscription, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	// Update subscription status
	subscription.Status = SubscriptionStatusDeleted

	// Remove from all indices
	delete(sm.subscriptions, subscriptionID)
	
	requestKey := sm.getRequestKey(subscription.NodeID, subscription.RequestID)
	delete(sm.subscriptionsByRequest, requestKey)

	// Remove from node index
	if nodeSubscriptions, exists := sm.subscriptionsByNode[subscription.NodeID]; exists {
		for i, sub := range nodeSubscriptions {
			if sub.SubscriptionID == subscriptionID {
				// Remove from slice
				sm.subscriptionsByNode[subscription.NodeID] = append(nodeSubscriptions[:i], nodeSubscriptions[i+1:]...)
				break
			}
		}
	}

	// Update metrics
	sm.metrics.E2Metrics.ActiveSubscriptions.Dec()

	sm.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
		"node_id":        subscription.NodeID,
	}).Info("RIC subscription deleted")

	// Send event
	sm.sendEvent(SubscriptionEvent{
		Type:         SubscriptionEventDeleted,
		Subscription: subscription,
		Timestamp:    time.Now(),
	})

	return nil
}

// ProcessSubscriptionDeleteResponse processes a subscription delete response
func (sm *SubscriptionManager) ProcessSubscriptionDeleteResponse(nodeID string, resp *RICSubscriptionDeleteResponse) error {
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
func (sm *SubscriptionManager) ProcessIndicationMessage(nodeID string, indication *RICIndication) error {
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
func (sm *SubscriptionManager) GetSubscription(subscriptionID string) (*RICSubscription, error) {
	sm.subscriptionsMutex.RLock()
	defer sm.subscriptionsMutex.RUnlock()

	subscription, exists := sm.subscriptions[subscriptionID]
	if !exists {
		return nil, fmt.Errorf("subscription %s not found", subscriptionID)
	}

	return subscription, nil
}

// GetSubscriptionsByNode retrieves all subscriptions for a node
func (sm *SubscriptionManager) GetSubscriptionsByNode(nodeID string) []*RICSubscription {
	sm.subscriptionsMutex.RLock()
	defer sm.subscriptionsMutex.RUnlock()

	subscriptions := sm.subscriptionsByNode[nodeID]
	if subscriptions == nil {
		return make([]*RICSubscription, 0)
	}

	// Return a copy to prevent external modification
	result := make([]*RICSubscription, len(subscriptions))
	copy(result, subscriptions)
	return result
}

// GetAllSubscriptions returns all active subscriptions
func (sm *SubscriptionManager) GetAllSubscriptions() []*RICSubscription {
	sm.subscriptionsMutex.RLock()
	defer sm.subscriptionsMutex.RUnlock()

	subscriptions := make([]*RICSubscription, 0, len(sm.subscriptions))
	for _, subscription := range sm.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions
}

// AddEventHandler adds a subscription event handler
func (sm *SubscriptionManager) AddEventHandler(handler SubscriptionEventHandler) {
	sm.eventHandlers = append(sm.eventHandlers, handler)
}

// validateSubscriptionRequest validates a subscription request
func (sm *SubscriptionManager) validateSubscriptionRequest(req *RICSubscriptionRequest) error {
	if req.RICRequestID.RICRequestorID < 0 {
		return fmt.Errorf("invalid RIC requestor ID: %d", req.RICRequestID.RICRequestorID)
	}

	if req.RICRequestID.RICInstanceID < 0 {
		return fmt.Errorf("invalid RIC instance ID: %d", req.RICRequestID.RICInstanceID)
	}

	if req.RANFunctionID < 0 {
		return fmt.Errorf("invalid RAN function ID: %d", req.RANFunctionID)
	}

	if len(req.RICSubscriptionDetails.RICEventTriggerDefinition) == 0 {
		return fmt.Errorf("RIC event trigger definition is required")
	}

	if len(req.RICSubscriptionDetails.RICActions) == 0 {
		return fmt.Errorf("at least one RIC action is required")
	}

	// Validate each action
	for i, action := range req.RICSubscriptionDetails.RICActions {
		if action.RICActionID < 0 {
			return fmt.Errorf("invalid RIC action ID at index %d: %d", i, action.RICActionID)
		}
	}

	return nil
}

// getRequestKey generates a unique key for a subscription request
func (sm *SubscriptionManager) getRequestKey(nodeID string, requestID RICRequestID) string {
	return fmt.Sprintf("%s_%d_%d", nodeID, requestID.RICRequestorID, requestID.RICInstanceID)
}

// sendEvent sends an event to all registered handlers
func (sm *SubscriptionManager) sendEvent(event SubscriptionEvent) {
	for _, handler := range sm.eventHandlers {
		go func(h SubscriptionEventHandler) {
			switch event.Type {
			case SubscriptionEventCreated:
				h.OnSubscriptionCreated(event.Subscription)
			case SubscriptionEventUpdated:
				h.OnSubscriptionUpdated(event.Subscription)
			case SubscriptionEventDeleted:
				h.OnSubscriptionDeleted(event.Subscription)
			case SubscriptionEventExpired:
				h.OnSubscriptionExpired(event.Subscription)
			case SubscriptionEventError:
				h.OnSubscriptionError(event.Subscription, event.Error)
			}
		}(handler)
	}
}

// expirationMonitor monitors subscription expiration
func (sm *SubscriptionManager) expirationMonitor() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()

	sm.logger.Debug("Starting subscription expiration monitor")

	for {
		select {
		case <-sm.ctx.Done():
			sm.logger.Debug("Subscription expiration monitor stopping")
			return
		case <-ticker.C:
			sm.checkExpiredSubscriptions()
		}
	}
}

// checkExpiredSubscriptions checks for and handles expired subscriptions
func (sm *SubscriptionManager) checkExpiredSubscriptions() {
	sm.subscriptionsMutex.RLock()
	expiredSubscriptions := make([]*RICSubscription, 0)
	
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
		subscription.Status = SubscriptionStatusExpired
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

// statisticsCollector periodically collects subscription statistics
func (sm *SubscriptionManager) statisticsCollector() {
	defer sm.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.collectStatistics()
		}
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
		case SubscriptionStatusActive:
			activeSubscriptions++
		case SubscriptionStatusPending:
			pendingSubscriptions++
		case SubscriptionStatusFailed:
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

// CleanupNodeSubscriptions removes all subscriptions for a disconnected node
func (sm *SubscriptionManager) CleanupNodeSubscriptions(nodeID string) error {
	subscriptions := sm.GetSubscriptionsByNode(nodeID)
	
	for _, subscription := range subscriptions {
		if err := sm.DeleteSubscription(subscription.SubscriptionID); err != nil {
			sm.logger.WithFields(logrus.Fields{
				"subscription_id": subscription.SubscriptionID,
				"node_id":        nodeID,
				"error":          err,
			}).Error("Failed to cleanup subscription")
		}
	}

	sm.logger.WithFields(logrus.Fields{
		"node_id":              nodeID,
		"cleaned_subscriptions": len(subscriptions),
	}).Info("Node subscriptions cleaned up")

	return nil
}