package xapp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// HealthMonitor monitors the health of xApp instances
type HealthMonitor struct {
	lifecycleManager *LifecycleManager
	logger           *logrus.Entry
	metrics          *monitoring.MetricsCollector

	// Health check configuration
	checkInterval    time.Duration
	checkTimeout     time.Duration
	failureThreshold int

	// Control and synchronization
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// Health check history
	healthHistory    map[XAppInstanceID][]HealthCheckResult
	historyMutex     sync.RWMutex
	
	// Event handlers
	eventHandlers []HealthEventHandler
}

// HealthEventHandler defines interface for health events
type HealthEventHandler interface {
	OnHealthCheckStarted(instanceID XAppInstanceID)
	OnHealthCheckCompleted(instanceID XAppInstanceID, result *HealthCheckResult)
	OnHealthStateChanged(instanceID XAppInstanceID, oldState, newState HealthState)
	OnHealthCheckFailed(instanceID XAppInstanceID, err error)
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(lm *LifecycleManager, baseLogger *logrus.Logger, metrics *monitoring.MetricsCollector) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	metrics.RegisterXAppMetrics()

	return &HealthMonitor{
		lifecycleManager: lm,
		logger:           baseLogger.WithField("component", "xapp-health-monitor"),
		metrics:          metrics,
		checkInterval:    30 * time.Second,
		checkTimeout:     10 * time.Second,
		failureThreshold: 3,
		ctx:              ctx,
		cancel:           cancel,
		healthHistory:    make(map[XAppInstanceID][]HealthCheckResult),
	}
}

// Start starts the health monitor
func (hm *HealthMonitor) Start(ctx context.Context) error {
	hm.logger.Info("Starting xApp health monitor")

	// Start health check worker
	hm.wg.Add(1)
	go hm.healthCheckWorker()

	hm.logger.Info("xApp health monitor started successfully")
	return nil
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop(ctx context.Context) error {
	hm.logger.Info("Stopping xApp health monitor")

	hm.cancel()

	done := make(chan struct{})
	go func() {
		hm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		hm.logger.Info("xApp health monitor stopped successfully")
	case <-ctx.Done():
		hm.logger.Warn("xApp health monitor shutdown timeout")
	}

	return nil
}

// PerformHealthCheck performs a health check on a specific instance
func (hm *HealthMonitor) PerformHealthCheck(instanceID XAppInstanceID) (*HealthCheckResult, error) {
	startTime := time.Now()

	// Send event
	for _, handler := range hm.eventHandlers {
		go handler.OnHealthCheckStarted(instanceID)
	}

	// Get instance from lifecycle manager
	instance, err := hm.lifecycleManager.GetInstance(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Skip health check for non-running instances
	if instance.Status != XAppStatusRunning {
		return &HealthCheckResult{
			Timestamp: startTime,
			State:     HealthStateUnknown,
			Message:   fmt.Sprintf("Instance is not running (status: %s)", instance.Status),
			Duration:  time.Since(startTime),
		}, nil
	}

	// Perform health check through deployment engine
	ctx, cancel := context.WithTimeout(hm.ctx, hm.checkTimeout)
	defer cancel()

	healthStatus, err := hm.lifecycleManager.deploymentEngine.PerformHealthCheck(ctx, instanceID)
	if err != nil {
		result := &HealthCheckResult{
			Timestamp: startTime,
			State:     HealthStateUnhealthy,
			Message:   fmt.Sprintf("Health check failed: %v", err),
			Duration:  time.Since(startTime),
		}

		// Send failed event
		for _, handler := range hm.eventHandlers {
			go handler.OnHealthCheckFailed(instanceID, err)
		}

		return result, err
	}

	// Create result
	result := &HealthCheckResult{
		Timestamp: startTime,
		State:     healthStatus.Overall,
		Message:   "Health check completed",
		Duration:  time.Since(startTime),
	}

	// Update instance health status
	hm.updateInstanceHealth(instanceID, healthStatus, result)

	// Send completion event
	for _, handler := range hm.eventHandlers {
		go handler.OnHealthCheckCompleted(instanceID, result)
	}

	// Update metrics
	hm.metrics.XAppMetrics.HealthChecksTotal.WithLabelValues(string(result.State)).Inc()

	hm.logger.WithFields(logrus.Fields{
		"instance_id": instanceID,
		"state":       result.State,
		"duration":    result.Duration,
	}).Debug("Health check completed")

	return result, nil
}

// GetHealthHistory returns health check history for an instance
func (hm *HealthMonitor) GetHealthHistory(instanceID XAppInstanceID, limit int) []HealthCheckResult {
	hm.historyMutex.RLock()
	defer hm.historyMutex.RUnlock()

	history, exists := hm.healthHistory[instanceID]
	if !exists {
		return nil
	}

	// Return last 'limit' entries
	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:]
	}

	return history
}

// GetOverallHealth returns overall health status for all instances
func (hm *HealthMonitor) GetOverallHealth() map[string]interface{} {
	instances := hm.lifecycleManager.ListInstances()
	
	var healthyCount, unhealthyCount, unknownCount, degradedCount int
	
	for _, instance := range instances {
		switch instance.HealthStatus.Overall {
		case HealthStateHealthy:
			healthyCount++
		case HealthStateUnhealthy:
			unhealthyCount++
		case HealthStateDegraded:
			degradedCount++
		default:
			unknownCount++
		}
	}

	totalInstances := len(instances)
	healthPercentage := 0.0
	if totalInstances > 0 {
		healthPercentage = float64(healthyCount) / float64(totalInstances) * 100
	}

	return map[string]interface{}{
		"total_instances":    totalInstances,
		"healthy_instances":  healthyCount,
		"unhealthy_instances": unhealthyCount,
		"degraded_instances": degradedCount,
		"unknown_instances":  unknownCount,
		"health_percentage":  healthPercentage,
		"overall_status":     hm.determineOverallStatus(healthyCount, unhealthyCount, degradedCount, unknownCount),
	}
}

// Background workers

func (hm *HealthMonitor) healthCheckWorker() {
	defer hm.wg.Done()

	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			hm.performHealthChecks()
		}
	}
}

func (hm *HealthMonitor) performHealthChecks() {
	instances := hm.lifecycleManager.ListInstances()

	// Perform health checks concurrently
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent checks

	for _, instance := range instances {
		// Only check running instances
		if instance.Status != XAppStatusRunning {
			continue
		}

		wg.Add(1)
		go func(instanceID XAppInstanceID) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := hm.PerformHealthCheck(instanceID)
			if err != nil {
				hm.logger.WithError(err).WithField("instance_id", instanceID).Error("Health check failed")
				
				// Handle consecutive failures
				hm.handleHealthCheckFailure(instanceID, err)
			} else {
				// Record successful health check
				hm.recordHealthCheck(instanceID, result)
			}
		}(instance.InstanceID)
	}

	wg.Wait()
}

func (hm *HealthMonitor) updateInstanceHealth(instanceID XAppInstanceID, healthStatus *XAppHealthStatus, result *HealthCheckResult) {
	// Get current instance to check for state changes
	instance, err := hm.lifecycleManager.GetInstance(instanceID)
	if err != nil {
		hm.logger.WithError(err).WithField("instance_id", instanceID).Error("Failed to get instance for health update")
		return
	}

	oldState := instance.HealthStatus.Overall
	newState := healthStatus.Overall

	// Update instance health in lifecycle manager
	hm.lifecycleManager.xappsMutex.Lock()
	if storedInstance, exists := hm.lifecycleManager.instances[instanceID]; exists {
		storedInstance.HealthStatus = *healthStatus
		storedInstance.HealthStatus.LastHealthCheck = result.Timestamp
		
		// Add to health history
		if len(storedInstance.HealthStatus.HealthHistory) >= 100 {
			// Keep only last 99 entries
			storedInstance.HealthStatus.HealthHistory = storedInstance.HealthStatus.HealthHistory[1:]
		}
		storedInstance.HealthStatus.HealthHistory = append(storedInstance.HealthStatus.HealthHistory, *result)
		
		storedInstance.UpdatedAt = time.Now()
	}
	hm.lifecycleManager.xappsMutex.Unlock()

	// Send state change event if state changed
	if oldState != newState {
		for _, handler := range hm.eventHandlers {
			go handler.OnHealthStateChanged(instanceID, oldState, newState)
		}

		// Send lifecycle manager event for health changes
		for _, handler := range hm.lifecycleManager.eventHandlers {
			go handler.OnHealthChanged(instanceID, oldState, newState)
		}
	}
}

func (hm *HealthMonitor) recordHealthCheck(instanceID XAppInstanceID, result *HealthCheckResult) {
	hm.historyMutex.Lock()
	defer hm.historyMutex.Unlock()

	history := hm.healthHistory[instanceID]
	
	// Keep only last 100 entries
	if len(history) >= 100 {
		history = history[1:]
	}
	
	history = append(history, *result)
	hm.healthHistory[instanceID] = history
}

func (hm *HealthMonitor) handleHealthCheckFailure(instanceID XAppInstanceID, err error) {
	// Get failure count from history
	history := hm.GetHealthHistory(instanceID, hm.failureThreshold)
	
	failureCount := 0
	for _, check := range history {
		if check.State == HealthStateUnhealthy {
			failureCount++
		}
	}

	// If we've exceeded the failure threshold, mark instance as failed
	if failureCount >= hm.failureThreshold {
		hm.logger.WithFields(logrus.Fields{
			"instance_id":     instanceID,
			"failure_count":   failureCount,
			"threshold":       hm.failureThreshold,
		}).Error("Instance exceeded health check failure threshold")

		// Update instance status to failed
		hm.lifecycleManager.updateInstanceStatus(instanceID, XAppStatusFailed, LifecycleInactive)
		hm.lifecycleManager.updateInstanceError(instanceID, fmt.Sprintf("Health check failures: %v", err))

		// Send failure event
		instance, _ := hm.lifecycleManager.GetInstance(instanceID)
		if instance != nil {
			for _, handler := range hm.lifecycleManager.eventHandlers {
				go handler.OnInstanceFailed(instance, err)
			}
		}
	}
}

func (hm *HealthMonitor) determineOverallStatus(healthy, unhealthy, degraded, unknown int) string {
	total := healthy + unhealthy + degraded + unknown
	
	if total == 0 {
		return "unknown"
	}
	
	// If more than 50% are healthy, overall is healthy
	if float64(healthy)/float64(total) > 0.5 {
		return "healthy"
	}
	
	// If any are unhealthy, overall is unhealthy
	if unhealthy > 0 {
		return "unhealthy"
	}
	
	// If any are degraded, overall is degraded
	if degraded > 0 {
		return "degraded"
	}
	
	return "unknown"
}

// Configuration methods

// SetCheckInterval sets the health check interval
func (hm *HealthMonitor) SetCheckInterval(interval time.Duration) {
	hm.checkInterval = interval
	hm.logger.WithField("interval", interval).Info("Health check interval updated")
}

// SetCheckTimeout sets the health check timeout
func (hm *HealthMonitor) SetCheckTimeout(timeout time.Duration) {
	hm.checkTimeout = timeout
	hm.logger.WithField("timeout", timeout).Info("Health check timeout updated")
}

// SetFailureThreshold sets the failure threshold
func (hm *HealthMonitor) SetFailureThreshold(threshold int) {
	hm.failureThreshold = threshold
	hm.logger.WithField("threshold", threshold).Info("Health check failure threshold updated")
}

// Public interface methods

// AddEventHandler adds a health event handler
func (hm *HealthMonitor) AddEventHandler(handler HealthEventHandler) {
	hm.eventHandlers = append(hm.eventHandlers, handler)
}

// GetHealthStats returns health monitoring statistics
func (hm *HealthMonitor) GetHealthStats() map[string]interface{} {
	return map[string]interface{}{
		"check_interval":     hm.checkInterval,
		"check_timeout":      hm.checkTimeout,
		"failure_threshold":  hm.failureThreshold,
		"overall_health":     hm.GetOverallHealth(),
		"total_checks":       len(hm.healthHistory),
	}
}