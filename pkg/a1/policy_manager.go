package a1

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
	"github.com/xeipuuv/gojsonschema"
)

// PolicyManager manages A1 policy types and instances
type PolicyManager struct {
	config  *config.A1Config
	logger  *logrus.Logger
	metrics *monitoring.MetricsCollector
	db      *pgxpool.Pool

	// Event handling
	eventHandlers []PolicyEventHandler

	// Background tasks
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Configuration
	maxPolicyTypes     int
	maxPolicyInstances int
	policyTimeout      time.Duration
}

// PolicyEventHandler defines interface for handling policy events
type PolicyEventHandler interface {
	OnPolicyTypeCreated(policyType *PolicyType)
	OnPolicyTypeDeleted(policyTypeID PolicyTypeID)
	OnPolicyInstanceCreated(policy *PolicyInstance)
	OnPolicyInstanceUpdated(policy *PolicyInstance)
	OnPolicyInstanceDeleted(policyID PolicyID)
	OnPolicyEnforced(policy *PolicyInstance)
	OnPolicyFailed(policy *PolicyInstance, err error)
}

// NewPolicyManager creates a new policy manager
func NewPolicyManager(config *config.A1Config, logger *logrus.Logger, metrics *monitoring.MetricsCollector, db *pgxpool.Pool) *PolicyManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &PolicyManager{
		config:             config,
		logger:             logger.WithField("component", "policy-manager"),
		metrics:            metrics,
		db:                 db,
		ctx:                ctx,
		cancel:             cancel,
		maxPolicyTypes:     100,  // Default limit
		maxPolicyInstances: 1000, // Default limit
		policyTimeout:      30 * time.Second,
	}
}

// Start starts the policy manager
func (pm *PolicyManager) Start(ctx context.Context) error {
	pm.logger.Info("Starting A1 Policy Manager")

	// Start background workers
	pm.wg.Add(2)
	go pm.policyEnforcementWorker()
	go pm.statisticsCollector()

	pm.logger.Info("A1 Policy Manager started successfully")
	return nil
}

// Stop stops the policy manager
func (pm *PolicyManager) Stop(ctx context.Context) error {
	pm.logger.Info("Stopping A1 Policy Manager")

	// Cancel context to stop background workers
	pm.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		pm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		pm.logger.Info("A1 Policy Manager stopped successfully")
	case <-ctx.Done():
		pm.logger.Warn("A1 Policy Manager shutdown timeout")
	}

	return nil
}

// CreatePolicyType creates a new policy type
func (pm *PolicyManager) CreatePolicyType(req *PolicyTypeCreateRequest) (*PolicyType, error) {
	// Validate policy type ID
	if !req.PolicyTypeID.IsValid() {
		return nil, fmt.Errorf("invalid policy type ID")
	}

	// Validate policy schema
	if err := pm.validateSchema(req.PolicySchema); err != nil {
		return nil, fmt.Errorf("invalid policy schema: %w", err)
	}

	// Create policy type
	policyType := &PolicyType{
		PolicyTypeID: req.PolicyTypeID,
		Name:         req.Name,
		Description:  req.Description,
		PolicySchema: req.PolicySchema,
		CreateSchema: req.CreateSchema,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Store policy type in the database
	_, err := pm.db.Exec(pm.ctx,
		"INSERT INTO policy_types (id, name, description, schema, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		policyType.PolicyTypeID, policyType.Name, policyType.Description, policyType.PolicySchema, policyType.CreatedAt, policyType.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy type: %w", err)
	}

	pm.logger.WithFields(logrus.Fields{
		"policy_type_id": req.PolicyTypeID,
		"name":           req.Name,
	}).Info("Policy type created")

	// Send event
	pm.sendEvent(&PolicyEvent{
		EventID:      uuid.New().String(),
		EventType:    "POLICY_TYPE_CREATED",
		PolicyTypeID: req.PolicyTypeID,
		Timestamp:    time.Now(),
	})

	// Update metrics
	pm.metrics.A1Metrics.PolicyOperations.WithLabelValues("create", "policy_type", "success").Inc()

	return policyType, nil
}

// GetPolicyType retrieves a policy type by ID
func (pm *PolicyManager) GetPolicyType(policyTypeID PolicyTypeID) (*PolicyType, error) {
	policyType := &PolicyType{}
	err := pm.db.QueryRow(pm.ctx,
		"SELECT id, name, description, schema, created_at, updated_at FROM policy_types WHERE id = $1",
		policyTypeID).Scan(&policyType.PolicyTypeID, &policyType.Name, &policyType.Description, &policyType.PolicySchema, &policyType.CreatedAt, &policyType.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("policy type %s not found", policyTypeID)
	}

	return policyType, nil
}

// GetAllPolicyTypes returns all policy types
func (pm *PolicyManager) GetAllPolicyTypes() []*PolicyType {
	rows, err := pm.db.Query(pm.ctx, "SELECT id, name, description, schema, created_at, updated_at FROM policy_types")
	if err != nil {
		pm.logger.WithError(err).Error("Failed to get all policy types")
		return []*PolicyType{}
	}
	defer rows.Close()

	policyTypes := make([]*PolicyType, 0)
	for rows.Next() {
		policyType := &PolicyType{}
		err := rows.Scan(&policyType.PolicyTypeID, &policyType.Name, &policyType.Description, &policyType.PolicySchema, &policyType.CreatedAt, &policyType.UpdatedAt)
		if err != nil {
			pm.logger.WithError(err).Error("Failed to scan policy type")
			continue
		}
		policyTypes = append(policyTypes, policyType)
	}

	return policyTypes
}

// DeletePolicyType deletes a policy type
func (pm *PolicyManager) DeletePolicyType(policyTypeID PolicyTypeID) error {
	// Check if there are active policy instances of this type
	var count int
	err := pm.db.QueryRow(pm.ctx, "SELECT COUNT(*) FROM policy_instances WHERE policy_type_id = $1 AND status != 'DELETED'", policyTypeID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for active policy instances: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete policy type %s: active policy instances exist", policyTypeID)
	}

	// Delete policy type
	cmdTag, err := pm.db.Exec(pm.ctx, "DELETE FROM policy_types WHERE id = $1", policyTypeID)
	if err != nil {
		return fmt.Errorf("failed to delete policy type: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("policy type %s not found", policyTypeID)
	}

	pm.logger.WithField("policy_type_id", policyTypeID).Info("Policy type deleted")

	// Send event
	pm.sendEvent(&PolicyEvent{
		EventID:      uuid.New().String(),
		EventType:    "POLICY_TYPE_DELETED",
		PolicyTypeID: policyTypeID,
		Timestamp:    time.Now(),
	})

	// Update metrics
	pm.metrics.A1Metrics.PolicyOperations.WithLabelValues("delete", "policy_type", "success").Inc()

	return nil
}

// CreatePolicyInstance creates a new policy instance
func (pm *PolicyManager) CreatePolicyInstance(policyTypeID PolicyTypeID, policyID PolicyID, req *PolicyInstanceCreateRequest, userID string) (*PolicyInstance, error) {
	// Validate policy ID
	if !policyID.IsValid() {
		return nil, fmt.Errorf("invalid policy ID")
	}

	// Check if policy type exists
	policyType, err := pm.GetPolicyType(policyTypeID)
	if err != nil {
		return nil, err
	}

	// Validate policy data against schema
	if err := pm.validatePolicyData(req.PolicyData, policyType.PolicySchema); err != nil {
		return nil, fmt.Errorf("policy data validation failed: %w", err)
	}

	// Create policy instance
	policy := &PolicyInstance{
		PolicyID:        policyID,
		PolicyTypeID:    policyTypeID,
		PolicyData:      req.PolicyData,
		Status:          PolicyStatusNotEnforced,
		TargetNearRTRIC: req.TargetNearRTRIC,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		CreatedBy:       userID,
		UpdatedBy:       userID,
	}

	// Store policy instance in the database
	_, err = pm.db.Exec(pm.ctx,
		"INSERT INTO policy_instances (id, policy_type_id, policy_data, target, priority, description, status, created_at, updated_at, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		policy.PolicyID, policy.PolicyTypeID, policy.PolicyData, policy.TargetNearRTRIC, 0, "", policy.Status, policy.CreatedAt, policy.UpdatedAt, policy.CreatedBy, policy.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy instance: %w", err)
	}

	pm.logger.WithFields(logrus.Fields{
		"policy_id":      policyID,
		"policy_type_id": policyTypeID,
		"user_id":        userID,
		"target_ric":     req.TargetNearRTRIC,
	}).Info("Policy instance created")

	// Send event
	pm.sendEvent(&PolicyEvent{
		EventID:      uuid.New().String(),
		EventType:    PolicyEventCreated,
		PolicyID:     policyID,
		PolicyTypeID: policyTypeID,
		Timestamp:    time.Now(),
		UserID:       userID,
		NewStatus:    PolicyStatusNotEnforced,
	})

	// Update metrics
	pm.metrics.A1Metrics.PolicyOperations.WithLabelValues("create", "policy_instance", "success").Inc()
	pm.metrics.A1Metrics.PoliciesActive.Inc()

	// Trigger policy enforcement
	go pm.enforcePolicyAsync(policy)

	return policy, nil
}

// GetPolicyInstance retrieves a policy instance by ID
func (pm *PolicyManager) GetPolicyInstance(policyID PolicyID) (*PolicyInstance, error) {
	policy := &PolicyInstance{}
	err := pm.db.QueryRow(pm.ctx,
		"SELECT id, policy_type_id, policy_data, target, priority, description, status, created_at, updated_at, created_by, updated_by FROM policy_instances WHERE id = $1",
		policyID).Scan(&policy.PolicyID, &policy.PolicyTypeID, &policy.PolicyData, &policy.TargetNearRTRIC, &policy.Priority, &policy.Description, &policy.Status, &policy.CreatedAt, &policy.UpdatedAt, &policy.CreatedBy, &policy.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("policy instance %s not found", policyID)
	}

	return policy, nil
}

// GetPolicyInstancesByType returns all policy instances of a specific type
func (pm *PolicyManager) GetPolicyInstancesByType(policyTypeID PolicyTypeID) []*PolicyInstance {
	rows, err := pm.db.Query(pm.ctx, "SELECT id, policy_type_id, policy_data, target, priority, description, status, created_at, updated_at, created_by, updated_by FROM policy_instances WHERE policy_type_id = $1 AND status != 'DELETED'", policyTypeID)
	if err != nil {
		pm.logger.WithError(err).Error("Failed to get policy instances by type")
		return []*PolicyInstance{}
	}
	defer rows.Close()

	policies := make([]*PolicyInstance, 0)
	for rows.Next() {
		policy := &PolicyInstance{}
		err := rows.Scan(&policy.PolicyID, &policy.PolicyTypeID, &policy.PolicyData, &policy.TargetNearRTRIC, &policy.Priority, &policy.Description, &policy.Status, &policy.CreatedAt, &policy.UpdatedAt, &policy.CreatedBy, &policy.UpdatedBy)
		if err != nil {
			pm.logger.WithError(err).Error("Failed to scan policy instance")
			continue
		}
		policies = append(policies, policy)
	}

	return policies
}

// GetAllPolicyInstances returns all policy instances
func (pm *PolicyManager) GetAllPolicyInstances() []*PolicyInstance {
	rows, err := pm.db.Query(pm.ctx, "SELECT id, policy_type_id, policy_data, target, priority, description, status, created_at, updated_at, created_by, updated_by FROM policy_instances WHERE status != 'DELETED'")
	if err != nil {
		pm.logger.WithError(err).Error("Failed to get all policy instances")
		return []*PolicyInstance{}
	}
	defer rows.Close()

	policies := make([]*PolicyInstance, 0)
	for rows.Next() {
		policy := &PolicyInstance{}
		err := rows.Scan(&policy.PolicyID, &policy.PolicyTypeID, &policy.PolicyData, &policy.TargetNearRTRIC, &policy.Priority, &policy.Description, &policy.Status, &policy.CreatedAt, &policy.UpdatedAt, &policy.CreatedBy, &policy.UpdatedBy)
		if err != nil {
			pm.logger.WithError(err).Error("Failed to scan policy instance")
			continue
		}
		policies = append(policies, policy)
	}

	return policies
}

// UpdatePolicyInstance updates an existing policy instance
func (pm *PolicyManager) UpdatePolicyInstance(policyID PolicyID, req *PolicyInstanceUpdateRequest, userID string) (*PolicyInstance, error) {
	// Get existing policy instance
	policy, err := pm.GetPolicyInstance(policyID)
	if err != nil {
		return nil, err
	}

	// Check if policy is in a state that allows updates
	if policy.Status == PolicyStatusDeleted {
		return nil, fmt.Errorf("cannot update deleted policy")
	}

	// Get policy type for validation
	policyType, err := pm.GetPolicyType(policy.PolicyTypeID)
	if err != nil {
		return nil, err
	}

	// Validate new policy data
	if err := pm.validatePolicyData(req.PolicyData, policyType.PolicySchema); err != nil {
		return nil, fmt.Errorf("policy data validation failed: %w", err)
	}

	// Store old status for event
	oldStatus := policy.Status

	// Update policy instance fields
	policy.PolicyData = req.PolicyData
	policy.TargetNearRTRIC = req.TargetNearRTRIC
	policy.Status = PolicyStatusNotEnforced // Reset to not enforced after update
	policy.UpdatedAt = time.Now()
	policy.UpdatedBy = userID
	policy.EnforcedAt = nil

	// Update policy instance in the database
	_, err = pm.db.Exec(pm.ctx,
		"UPDATE policy_instances SET policy_data = $1, target = $2, status = $3, updated_at = $4, updated_by = $5 WHERE id = $6",
		policy.PolicyData, policy.TargetNearRTRIC, policy.Status, policy.UpdatedAt, policy.UpdatedBy, policy.PolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to update policy instance: %w", err)
	}

	pm.logger.WithFields(logrus.Fields{
		"policy_id":      policyID,
		"policy_type_id": policy.PolicyTypeID,
		"user_id":        userID,
		"old_status":     oldStatus,
		"new_status":     policy.Status,
	}).Info("Policy instance updated")

	// Send event
	pm.sendEvent(&PolicyEvent{
		EventID:      uuid.New().String(),
		EventType:    PolicyEventUpdated,
		PolicyID:     policyID,
		PolicyTypeID: policy.PolicyTypeID,
		Timestamp:    time.Now(),
		UserID:       userID,
		OldStatus:    oldStatus,
		NewStatus:    policy.Status,
	})

	// Update metrics
	pm.metrics.A1Metrics.PolicyOperations.WithLabelValues("update", "policy_instance", "success").Inc()

	// Trigger policy enforcement
	go pm.enforcePolicyAsync(policy)

	return policy, nil
}

// DeletePolicyInstance deletes a policy instance
func (pm *PolicyManager) DeletePolicyInstance(policyID PolicyID, userID string) error {
	// Get existing policy instance to get old status
	policy, err := pm.GetPolicyInstance(policyID)
	if err != nil {
		return err
	}

	// Check if policy is already deleted
	if policy.Status == PolicyStatusDeleted {
		return fmt.Errorf("policy instance %s is already deleted", policyID)
	}

	oldStatus := policy.Status

	// Mark policy as deleted in the database
	_, err = pm.db.Exec(pm.ctx, "UPDATE policy_instances SET status = $1, updated_at = $2, updated_by = $3 WHERE id = $4",
		PolicyStatusDeleted, time.Now(), userID, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy instance: %w", err)
	}

	pm.logger.WithFields(logrus.Fields{
		"policy_id":      policyID,
		"policy_type_id": policy.PolicyTypeID,
		"user_id":        userID,
		"old_status":     oldStatus,
	}).Info("Policy instance deleted")

	// Send event
	pm.sendEvent(&PolicyEvent{
		EventID:      uuid.New().String(),
		EventType:    PolicyEventDeleted,
		PolicyID:     policyID,
		PolicyTypeID: policy.PolicyTypeID,
		Timestamp:    time.Now(),
		UserID:       userID,
		OldStatus:    oldStatus,
		NewStatus:    PolicyStatusDeleted,
	})

	// Update metrics
	pm.metrics.A1Metrics.PolicyOperations.WithLabelValues("delete", "policy_instance", "success").Inc()
	pm.metrics.A1Metrics.PoliciesActive.Dec()

	return nil
}

// GetPolicyStatus returns the status of a policy instance
func (pm *PolicyManager) GetPolicyStatus(policyID PolicyID) (*PolicyStatusInfo, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	policy, exists := pm.policyInstances[policyID]
	if !exists {
		return nil, fmt.Errorf("policy instance %s not found", policyID)
	}

	statusInfo := &PolicyStatusInfo{
		PolicyID:     policy.PolicyID,
		PolicyTypeID: policy.PolicyTypeID,
		Status:       policy.Status,
		StatusReason: policy.StatusReason,
		LastUpdate:   policy.UpdatedAt,
	}

	return statusInfo, nil
}

// validateSchema validates a JSON schema
func (pm *PolicyManager) validateSchema(schema interface{}) error {
	// Convert schema to JSON for validation
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Validate that it's a valid JSON schema
	loader := gojsonschema.NewBytesLoader(schemaBytes)
	_, err = gojsonschema.NewSchema(loader)
	if err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	return nil
}

// validatePolicyData validates policy data against a schema
func (pm *PolicyManager) validatePolicyData(data interface{}, schema interface{}) error {
	// Convert schema and data to JSON for validation
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal policy data: %w", err)
	}

	// Create JSON schema validator
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	dataLoader := gojsonschema.NewBytesLoader(dataBytes)

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// enforcePolicyAsync enforces a policy asynchronously
func (pm *PolicyManager) enforcePolicyAsync(policy *PolicyInstance) {
	pm.logger.WithFields(logrus.Fields{
		"policy_id":      policy.PolicyID,
		"policy_type_id": policy.PolicyTypeID,
		"target_ric":     policy.TargetNearRTRIC,
	}).Info("Starting policy enforcement")

	// Simulate policy enforcement (in real implementation, this would communicate with Near-RT RIC)
	time.Sleep(100 * time.Millisecond) // Simulate processing time

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Check if policy still exists and is in correct state
	currentPolicy, exists := pm.policyInstances[policy.PolicyID]
	if !exists || currentPolicy.Status == PolicyStatusDeleted {
		pm.logger.WithField("policy_id", policy.PolicyID).Debug("Policy no longer exists or was deleted")
		return
	}

	// Simulate enforcement result (90% success rate)
	if time.Now().UnixNano()%10 < 9 {
		// Success
		currentPolicy.Status = PolicyStatusEnforced
		currentPolicy.StatusReason = "Policy successfully enforced"
		now := time.Now()
		currentPolicy.EnforcedAt = &now
		currentPolicy.UpdatedAt = now

		pm.logger.WithField("policy_id", policy.PolicyID).Info("Policy enforcement successful")

		// Send event
		pm.sendEvent(&PolicyEvent{
			EventID:      uuid.New().String(),
			EventType:    PolicyEventEnforced,
			PolicyID:     policy.PolicyID,
			PolicyTypeID: policy.PolicyTypeID,
			Timestamp:    time.Now(),
			OldStatus:    PolicyStatusNotEnforced,
			NewStatus:    PolicyStatusEnforced,
		})
	} else {
		// Failure
		currentPolicy.StatusReason = "Policy enforcement failed"
		currentPolicy.UpdatedAt = time.Now()

		pm.logger.WithField("policy_id", policy.PolicyID).Warn("Policy enforcement failed")

		// Send event
		pm.sendEvent(&PolicyEvent{
			EventID:      uuid.New().String(),
			EventType:    PolicyEventFailed,
			PolicyID:     policy.PolicyID,
			PolicyTypeID: policy.PolicyTypeID,
			Timestamp:    time.Now(),
			Details: map[string]interface{}{
				"error": "Simulated enforcement failure",
			},
		})

		// Update metrics
		pm.metrics.A1Metrics.PolicyErrors.Inc()
	}
}

// policyEnforcementWorker periodically checks for policies that need enforcement
func (pm *PolicyManager) policyEnforcementWorker() {
	defer pm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	pm.logger.Debug("Starting policy enforcement worker")

	for {
		select {
		case <-pm.ctx.Done():
			pm.logger.Debug("Policy enforcement worker stopping")
			return
		case <-ticker.C:
			pm.checkPolicyEnforcement()
		}
	}
}

// checkPolicyEnforcement checks for policies that need re-enforcement
func (pm *PolicyManager) checkPolicyEnforcement() {
	pm.mutex.RLock()
	policiesToCheck := make([]*PolicyInstance, 0)
	
	for _, policy := range pm.policyInstances {
		// Check for policies that have been not enforced for too long
		if policy.Status == PolicyStatusNotEnforced && time.Since(policy.UpdatedAt) > pm.policyTimeout {
			policiesToCheck = append(policiesToCheck, policy)
		}
	}
	pm.mutex.RUnlock()

	for _, policy := range policiesToCheck {
		pm.logger.WithFields(logrus.Fields{
			"policy_id":      policy.PolicyID,
			"not_enforced_since": policy.UpdatedAt,
		}).Debug("Retrying policy enforcement")
		
		go pm.enforcePolicyAsync(policy)
	}
}

// statisticsCollector periodically collects policy statistics
func (pm *PolicyManager) statisticsCollector() {
	defer pm.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.collectStatistics()
		}
	}
}

// collectStatistics collects and updates policy statistics
func (pm *PolicyManager) collectStatistics() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	totalTypes := len(pm.policyTypes)
	totalInstances := 0
	enforcedPolicies := 0
	failedPolicies := 0

	for _, policy := range pm.policyInstances {
		if policy.Status != PolicyStatusDeleted {
			totalInstances++
			if policy.Status == PolicyStatusEnforced {
				enforcedPolicies++
			} else if policy.StatusReason != "" {
				failedPolicies++
			}
		}
	}

	pm.logger.WithFields(logrus.Fields{
		"total_policy_types":     totalTypes,
		"total_policy_instances": totalInstances,
		"enforced_policies":      enforcedPolicies,
		"failed_policies":        failedPolicies,
	}).Debug("Policy statistics collected")
}

// AddEventHandler adds a policy event handler
func (pm *PolicyManager) AddEventHandler(handler PolicyEventHandler) {
	pm.eventHandlers = append(pm.eventHandlers, handler)
}

// sendEvent sends a policy event to all registered handlers
func (pm *PolicyManager) sendEvent(event *PolicyEvent) {
	for _, handler := range pm.eventHandlers {
		go func(h PolicyEventHandler) {
			switch event.EventType {
			case PolicyEventCreated:
				if policy, exists := pm.policyInstances[event.PolicyID]; exists {
					h.OnPolicyInstanceCreated(policy)
				}
			case PolicyEventUpdated:
				if policy, exists := pm.policyInstances[event.PolicyID]; exists {
					h.OnPolicyInstanceUpdated(policy)
				}
			case PolicyEventDeleted:
				h.OnPolicyInstanceDeleted(event.PolicyID)
			case PolicyEventEnforced:
				if policy, exists := pm.policyInstances[event.PolicyID]; exists {
					h.OnPolicyEnforced(policy)
				}
			case PolicyEventFailed:
				if policy, exists := pm.policyInstances[event.PolicyID]; exists {
					h.OnPolicyFailed(policy, fmt.Errorf("policy enforcement failed"))
				}
			}
		}(handler)
	}
}

// GetStatistics returns current policy manager statistics
func (pm *PolicyManager) GetStatistics() *A1Statistics {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	totalTypes := len(pm.policyTypes)
	totalInstances := 0
	enforcedPolicies := 0
	failedPolicies := 0

	for _, policy := range pm.policyInstances {
		if policy.Status != PolicyStatusDeleted {
			totalInstances++
			if policy.Status == PolicyStatusEnforced {
				enforcedPolicies++
			} else if policy.StatusReason != "" {
				failedPolicies++
			}
		}
	}

	return &A1Statistics{
		TotalPolicyTypes:     totalTypes,
		TotalPolicyInstances: totalInstances,
		EnforcedPolicies:     enforcedPolicies,
		FailedPolicies:       failedPolicies,
		LastUpdate:          time.Now(),
	}
}