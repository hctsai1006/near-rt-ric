package xapp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// LifecycleManager manages the lifecycle of xApps
type LifecycleManager struct {
	logger  *logrus.Logger
	metrics *monitoring.MetricsCollector

	// xApp registry
	xapps      map[XAppID]*XApp
	instances  map[XAppInstanceID]*XAppInstance
	xappsMutex sync.RWMutex

	// Deployment engine
	deploymentEngine DeploymentEngine

	// Health monitoring
	healthMonitor *HealthMonitor

	// Dependency resolver
	dependencyResolver *DependencyResolver

	// Event handlers
	eventHandlers []XAppEventHandler

	// Background workers
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Configuration
	maxInstances        int
	deploymentTimeout   time.Duration
	healthCheckInterval time.Duration
}

// XAppEventHandler defines interface for xApp events
type XAppEventHandler interface {
	OnXAppCreated(xapp *XApp)
	OnXAppUpdated(xapp *XApp)
	OnXAppDeleted(xappID XAppID)
	OnInstanceCreated(instance *XAppInstance)
	OnInstanceStarted(instance *XAppInstance)
	OnInstanceStopped(instance *XAppInstance)
	OnInstanceFailed(instance *XAppInstance, err error)
	OnInstanceDeleted(instanceID XAppInstanceID)
	OnHealthChanged(instanceID XAppInstanceID, oldState, newState HealthState)
}

// DeploymentEngine interface for different deployment backends
type DeploymentEngine interface {
	Deploy(ctx context.Context, xapp *XApp, config map[string]interface{}) (*XAppInstance, error)
	Update(ctx context.Context, instance *XAppInstance, config map[string]interface{}) error
	Stop(ctx context.Context, instanceID XAppInstanceID) error
	Delete(ctx context.Context, instanceID XAppInstanceID) error
	GetRuntimeInfo(ctx context.Context, instanceID XAppInstanceID) (*XAppRuntimeInfo, error)
	GetMetrics(ctx context.Context, instanceID XAppInstanceID) (*XAppMetrics, error)
	PerformHealthCheck(ctx context.Context, instanceID XAppInstanceID) (*XAppHealthStatus, error)
}

// NewLifecycleManager creates a new xApp lifecycle manager
func NewLifecycleManager(logger *logrus.Logger, metrics *monitoring.MetricsCollector, deploymentEngine DeploymentEngine) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())

	lm := &LifecycleManager{
		logger:              logger,
		metrics:             metrics,
		xapps:               make(map[XAppID]*XApp),
		instances:           make(map[XAppInstanceID]*XAppInstance),
		deploymentEngine:    deploymentEngine,
		ctx:                 ctx,
		cancel:              cancel,
		maxInstances:        1000, // Default limit
		deploymentTimeout:   5 * time.Minute,
		healthCheckInterval: 30 * time.Second,
	}

	// Initialize health monitor
	lm.healthMonitor = NewHealthMonitor(lm, logger, metrics)

	// Initialize dependency resolver
	lm.dependencyResolver = NewDependencyResolver(lm, logger)

	lm.logger.Info("xApp lifecycle manager initialized")
	return lm
}

// Start starts the lifecycle manager
func (lm *LifecycleManager) Start(ctx context.Context) error {
	lm.logger.Info("Starting xApp lifecycle manager")

	// Start health monitor
	if err := lm.healthMonitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}

	// Start background workers
	lm.wg.Add(2)
	go lm.metricsCollector()
	go lm.instanceMonitor()

	lm.logger.Info("xApp lifecycle manager started successfully")
	return nil
}

// Stop stops the lifecycle manager gracefully
func (lm *LifecycleManager) Stop(ctx context.Context) error {
	lm.logger.Info("Stopping xApp lifecycle manager")

	// Stop health monitor
	if err := lm.healthMonitor.Stop(ctx); err != nil {
		lm.logger.WithError(err).Error("Error stopping health monitor")
	}

	// Cancel context and wait for workers
	lm.cancel()

	done := make(chan struct{})
	go func() {
		lm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		lm.logger.Info("xApp lifecycle manager stopped successfully")
	case <-ctx.Done():
		lm.logger.Warn("xApp lifecycle manager shutdown timeout")
	}

	return nil
}

// XApp Management

// RegisterXApp registers a new xApp in the platform
func (lm *LifecycleManager) RegisterXApp(xapp *XApp) error {
	lm.xappsMutex.Lock()
	defer lm.xappsMutex.Unlock()

	// Validate xApp
	if err := lm.validateXApp(xapp); err != nil {
		return fmt.Errorf("xApp validation failed: %w", err)
	}

	// Check if xApp already exists
	if _, exists := lm.xapps[xapp.XAppID]; exists {
		return fmt.Errorf("xApp %s already exists", xapp.XAppID)
	}

	// Set timestamps
	now := time.Now()
	xapp.CreatedAt = now
	xapp.UpdatedAt = now

	// Store xApp
	lm.xapps[xapp.XAppID] = xapp

	lm.logger.WithFields(logrus.Fields{
		"xapp_id": xapp.XAppID,
		"name":    xapp.Name,
		"version": xapp.Version,
		"type":    xapp.Type,
	}).Info("xApp registered")

	// Send event
	lm.sendEvent(EventXAppCreated, xapp.XAppID, "", "xApp registered successfully", nil)

	// Update metrics
	lm.metrics.XAppMetrics.XAppsTotal.WithLabelValues(string(xapp.Type)).Inc()

	return nil
}

// UpdateXApp updates an existing xApp
func (lm *LifecycleManager) UpdateXApp(xappID XAppID, updatedXApp *XApp) error {
	lm.xappsMutex.Lock()
	defer lm.xappsMutex.Unlock()

	// Check if xApp exists
	existingXApp, exists := lm.xapps[xappID]
	if !exists {
		return fmt.Errorf("xApp %s not found", xappID)
	}

	// Validate updated xApp
	updatedXApp.XAppID = xappID
	if err := lm.validateXApp(updatedXApp); err != nil {
		return fmt.Errorf("xApp validation failed: %w", err)
	}

	// Update fields
	updatedXApp.CreatedAt = existingXApp.CreatedAt
	updatedXApp.UpdatedAt = time.Now()

	// Store updated xApp
	lm.xapps[xappID] = updatedXApp

	lm.logger.WithFields(logrus.Fields{
		"xapp_id": xappID,
		"name":    updatedXApp.Name,
		"version": updatedXApp.Version,
	}).Info("xApp updated")

	// Send event
	lm.sendEvent(EventXAppUpdated, xappID, "", "xApp updated successfully", nil)

	return nil
}

// UnregisterXApp removes an xApp from the platform
func (lm *LifecycleManager) UnregisterXApp(xappID XAppID) error {
	lm.xappsMutex.Lock()
	defer lm.xappsMutex.Unlock()

	// Check if xApp exists
	xapp, exists := lm.xapps[xappID]
	if !exists {
		return fmt.Errorf("xApp %s not found", xappID)
	}

	// Check if there are running instances
	runningInstances := lm.getInstancesByXApp(xappID)
	if len(runningInstances) > 0 {
		return fmt.Errorf("cannot unregister xApp %s: %d instances are still running", xappID, len(runningInstances))
	}

	// Remove xApp
	delete(lm.xapps, xappID)

	lm.logger.WithField("xapp_id", xappID).Info("xApp unregistered")

	// Send event
	lm.sendEvent(EventXAppDeleted, xappID, "", "xApp unregistered successfully", nil)

	// Update metrics
	lm.metrics.XAppMetrics.XAppsTotal.WithLabelValues(string(xapp.Type)).Dec()

	return nil
}

// GetXApp retrieves an xApp by ID
func (lm *LifecycleManager) GetXApp(xappID XAppID) (*XApp, error) {
	lm.xappsMutex.RLock()
	defer lm.xappsMutex.RUnlock()

	xapp, exists := lm.xapps[xappID]
	if !exists {
		return nil, fmt.Errorf("xApp %s not found", xappID)
	}

	return xapp, nil
}

// ListXApps returns all registered xApps
func (lm *LifecycleManager) ListXApps() []*XApp {
	lm.xappsMutex.RLock()
	defer lm.xappsMutex.RUnlock()

	xapps := make([]*XApp, 0, len(lm.xapps))
	for _, xapp := range lm.xapps {
		xapps = append(xapps, xapp)
	}

	return xapps
}

// Instance Management

// DeployInstance deploys a new instance of an xApp
func (lm *LifecycleManager) DeployInstance(xappID XAppID, config map[string]interface{}) (*XAppInstance, error) {
	lm.xappsMutex.RLock()
	xapp, exists := lm.xapps[xappID]
	lm.xappsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("xApp %s not found", xappID)
	}

	// Check instance limit
	if len(lm.instances) >= lm.maxInstances {
		return nil, fmt.Errorf("maximum number of instances (%d) reached", lm.maxInstances)
	}

	// Resolve dependencies
	if err := lm.dependencyResolver.ResolveDependencies(xapp); err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Create instance
	instanceID := XAppInstanceID(uuid.New().String())
	instance := &XAppInstance{
		InstanceID:     instanceID,
		XAppID:         xappID,
		Name:           fmt.Sprintf("%s-%s", xapp.Name, instanceID[:8]),
		Status:         XAppStatusPending,
		LifecycleState: LifecycleCreated,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Config:         config,
		HealthStatus: XAppHealthStatus{
			Overall:         HealthStateUnknown,
			LastHealthCheck: time.Now(),
		},
		Metrics: XAppMetrics{
			LastUpdated: time.Now(),
		},
	}

	// Store instance
	lm.xappsMutex.Lock()
	lm.instances[instanceID] = instance
	lm.xappsMutex.Unlock()

	lm.logger.WithFields(logrus.Fields{
		"instance_id": instanceID,
		"xapp_id":     xappID,
		"name":        instance.Name,
	}).Info("xApp instance created")

	// Send event
	lm.sendEvent(EventInstanceCreated, xappID, instanceID, "xApp instance created", nil)

	// Start deployment asynchronously
	go lm.deployInstanceAsync(instance, xapp, config)

	return instance, nil
}

// deployInstanceAsync performs asynchronous deployment
func (lm *LifecycleManager) deployInstanceAsync(instance *XAppInstance, xapp *XApp, config map[string]interface{}) {
	ctx, cancel := context.WithTimeout(lm.ctx, lm.deploymentTimeout)
	defer cancel()

	// Update status to deploying
	lm.updateInstanceStatus(instance.InstanceID, XAppStatusDeploying, LifecycleInstalled)

	// Deploy through deployment engine
	deployedInstance, err := lm.deploymentEngine.Deploy(ctx, xapp, config)
	if err != nil {
		lm.logger.WithError(err).WithField("instance_id", instance.InstanceID).Error("Deployment failed")

		// Update status to failed
		lm.updateInstanceStatus(instance.InstanceID, XAppStatusFailed, LifecycleInactive)
		lm.updateInstanceError(instance.InstanceID, err.Error())

		// Send event
		lm.sendEvent(EventInstanceFailed, instance.XAppID, instance.InstanceID, "Deployment failed", map[string]interface{}{
			"error": err.Error(),
		})

		// Update metrics
		lm.metrics.XAppMetrics.DeploymentErrors.Inc()

		return
	}

	// Update instance with deployment information
	lm.xappsMutex.Lock()
	if storedInstance, exists := lm.instances[instance.InstanceID]; exists {
		storedInstance.RuntimeInfo = deployedInstance.RuntimeInfo
		storedInstance.Status = XAppStatusRunning
		storedInstance.LifecycleState = LifecycleActive
		storedInstance.UpdatedAt = time.Now()
		now := time.Now()
		storedInstance.StartedAt = &now
		storedInstance.HealthStatus.Overall = HealthStateHealthy
	}
	lm.xappsMutex.Unlock()

	lm.logger.WithField("instance_id", instance.InstanceID).Info("xApp instance deployed successfully")

	// Send event
	lm.sendEvent(EventInstanceStarted, instance.XAppID, instance.InstanceID, "xApp instance started successfully", nil)

	// Update metrics
	lm.metrics.XAppMetrics.InstancesActive.Inc()
	lm.metrics.XAppMetrics.DeploymentSuccess.Inc()
}

// StopInstance stops a running xApp instance
func (lm *LifecycleManager) StopInstance(instanceID XAppInstanceID) error {
	lm.xappsMutex.RLock()
	instance, exists := lm.instances[instanceID]
	lm.xappsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	if instance.Status == XAppStatusStopped || instance.Status == XAppStatusStopping {
		return fmt.Errorf("instance %s is already stopped or stopping", instanceID)
	}

	// Update status to stopping
	lm.updateInstanceStatus(instanceID, XAppStatusStopping, LifecycleInactive)

	// Stop through deployment engine
	ctx, cancel := context.WithTimeout(lm.ctx, 30*time.Second)
	defer cancel()

	if err := lm.deploymentEngine.Stop(ctx, instanceID); err != nil {
		lm.logger.WithError(err).WithField("instance_id", instanceID).Error("Failed to stop instance")
		lm.updateInstanceError(instanceID, err.Error())
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	// Update status to stopped
	lm.updateInstanceStatus(instanceID, XAppStatusStopped, LifecycleInactive)

	// Update stopped time
	lm.xappsMutex.Lock()
	if storedInstance, exists := lm.instances[instanceID]; exists {
		now := time.Now()
		storedInstance.StoppedAt = &now
	}
	lm.xappsMutex.Unlock()

	lm.logger.WithField("instance_id", instanceID).Info("xApp instance stopped")

	// Send event
	lm.sendEvent(EventInstanceStopped, instance.XAppID, instanceID, "xApp instance stopped", nil)

	// Update metrics
	lm.metrics.XAppMetrics.InstancesActive.Dec()

	return nil
}

// DeleteInstance deletes an xApp instance
func (lm *LifecycleManager) DeleteInstance(instanceID XAppInstanceID) error {
	lm.xappsMutex.RLock()
	instance, exists := lm.instances[instanceID]
	lm.xappsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	// Stop instance if it's running
	if instance.Status == XAppStatusRunning {
		if err := lm.StopInstance(instanceID); err != nil {
			return fmt.Errorf("failed to stop instance before deletion: %w", err)
		}
	}

	// Delete through deployment engine
	ctx, cancel := context.WithTimeout(lm.ctx, 30*time.Second)
	defer cancel()

	if err := lm.deploymentEngine.Delete(ctx, instanceID); err != nil {
		lm.logger.WithError(err).WithField("instance_id", instanceID).Error("Failed to delete instance")
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	// Remove from instances map
	lm.xappsMutex.Lock()
	delete(lm.instances, instanceID)
	lm.xappsMutex.Unlock()

	lm.logger.WithField("instance_id", instanceID).Info("xApp instance deleted")

	// Send event
	lm.sendEvent(EventInstanceDeleted, instance.XAppID, instanceID, "xApp instance deleted", nil)

	return nil
}

// GetInstance retrieves an instance by ID
func (lm *LifecycleManager) GetInstance(instanceID XAppInstanceID) (*XAppInstance, error) {
	lm.xappsMutex.RLock()
	defer lm.xappsMutex.RUnlock()

	instance, exists := lm.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	return instance, nil
}

// ListInstances returns all instances
func (lm *LifecycleManager) ListInstances() []*XAppInstance {
	lm.xappsMutex.RLock()
	defer lm.xappsMutex.RUnlock()

	instances := make([]*XAppInstance, 0, len(lm.instances))
	for _, instance := range lm.instances {
		instances = append(instances, instance)
	}

	return instances
}

// GetInstancesByXApp returns instances for a specific xApp
func (lm *LifecycleManager) GetInstancesByXApp(xappID XAppID) []*XAppInstance {
	return lm.getInstancesByXApp(xappID)
}

// UpdateInstanceConfig updates the configuration of a running instance
func (lm *LifecycleManager) UpdateInstanceConfig(instanceID XAppInstanceID, config map[string]interface{}) error {
	lm.xappsMutex.RLock()
	instance, exists := lm.instances[instanceID]
	lm.xappsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	if instance.Status != XAppStatusRunning {
		return fmt.Errorf("can only update config of running instances")
	}

	// Update through deployment engine
	ctx, cancel := context.WithTimeout(lm.ctx, 30*time.Second)
	defer cancel()

	if err := lm.deploymentEngine.Update(ctx, instance, config); err != nil {
		return fmt.Errorf("failed to update instance config: %w", err)
	}

	// Update stored config
	lm.xappsMutex.Lock()
	if storedInstance, exists := lm.instances[instanceID]; exists {
		storedInstance.Config = config
		storedInstance.UpdatedAt = time.Now()
	}
	lm.xappsMutex.Unlock()

	lm.logger.WithField("instance_id", instanceID).Info("xApp instance config updated")

	// Send event
	lm.sendEvent(EventConfigChanged, instance.XAppID, instanceID, "Instance configuration updated", nil)

	return nil
}

// Helper methods

func (lm *LifecycleManager) validateXApp(xapp *XApp) error {
	if xapp.XAppID == "" {
		return fmt.Errorf("xApp ID is required")
	}

	if xapp.Name == "" {
		return fmt.Errorf("xApp name is required")
	}

	if xapp.Version == "" {
		return fmt.Errorf("xApp version is required")
	}

	if !xapp.Type.IsValid() {
		return fmt.Errorf("invalid xApp type: %s", xapp.Type)
	}

	if xapp.Deployment.Image == "" {
		return fmt.Errorf("container image is required")
	}

	return nil
}

func (lm *LifecycleManager) getInstancesByXApp(xappID XAppID) []*XAppInstance {
	lm.xappsMutex.RLock()
	defer lm.xappsMutex.RUnlock()

	var instances []*XAppInstance
	for _, instance := range lm.instances {
		if instance.XAppID == xappID {
			instances = append(instances, instance)
		}
	}

	return instances
}

func (lm *LifecycleManager) updateInstanceStatus(instanceID XAppInstanceID, status XAppStatus, lifecycleState XAppLifecycleState) {
	lm.xappsMutex.Lock()
	defer lm.xappsMutex.Unlock()

	if instance, exists := lm.instances[instanceID]; exists {
		oldStatus := instance.Status
		instance.Status = status
		instance.LifecycleState = lifecycleState
		instance.UpdatedAt = time.Now()

		lm.logger.WithFields(logrus.Fields{
			"instance_id": instanceID,
			"old_status":  oldStatus,
			"new_status":  status,
			"lifecycle":   lifecycleState,
		}).Debug("Instance status updated")
	}
}

func (lm *LifecycleManager) updateInstanceError(instanceID XAppInstanceID, errorMsg string) {
	lm.xappsMutex.Lock()
	defer lm.xappsMutex.Unlock()

	if instance, exists := lm.instances[instanceID]; exists {
		instance.LastError = errorMsg
		instance.ErrorCount++
		instance.UpdatedAt = time.Now()
	}
}

func (lm *LifecycleManager) sendEvent(eventType string, xappID XAppID, instanceID XAppInstanceID, message string, details map[string]interface{}) {
	_ = &XAppEvent{
		EventID:    uuid.New().String(),
		EventType:  eventType,
		XAppID:     xappID,
		InstanceID: instanceID,
		Timestamp:  time.Now(),
		Source:     "lifecycle-manager",
		Message:    message,
		Details:    details,
		Severity:   "info",
	}

	for _, handler := range lm.eventHandlers {
		go func(h XAppEventHandler) {
			switch eventType {
			case EventXAppCreated:
				if xapp, err := lm.GetXApp(xappID); err == nil {
					h.OnXAppCreated(xapp)
				}
			case EventXAppUpdated:
				if xapp, err := lm.GetXApp(xappID); err == nil {
					h.OnXAppUpdated(xapp)
				}
			case EventXAppDeleted:
				h.OnXAppDeleted(xappID)
			case EventInstanceCreated:
				if instance, err := lm.GetInstance(instanceID); err == nil {
					h.OnInstanceCreated(instance)
				}
			case EventInstanceStarted:
				if instance, err := lm.GetInstance(instanceID); err == nil {
					h.OnInstanceStarted(instance)
				}
			case EventInstanceStopped:
				if instance, err := lm.GetInstance(instanceID); err == nil {
					h.OnInstanceStopped(instance)
				}
			case EventInstanceFailed:
				if instance, err := lm.GetInstance(instanceID); err == nil {
					h.OnInstanceFailed(instance, fmt.Errorf("%s", message))
				}
			case EventInstanceDeleted:
				h.OnInstanceDeleted(instanceID)
			}
		}(handler)
	}
}

// Background workers

func (lm *LifecycleManager) metricsCollector() {
	defer lm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-ticker.C:
			lm.collectMetrics()
		}
	}
}

func (lm *LifecycleManager) collectMetrics() {
	lm.xappsMutex.RLock()
	defer lm.xappsMutex.RUnlock()

	var runningCount, stoppedCount, failedCount int

	for _, instance := range lm.instances {
		switch instance.Status {
		case XAppStatusRunning:
			runningCount++
		case XAppStatusStopped:
			stoppedCount++
		case XAppStatusFailed:
			failedCount++
		}
	}

	// Update Prometheus metrics
	lm.metrics.XAppMetrics.InstancesActive.Set(float64(runningCount))
	lm.metrics.XAppMetrics.InstancesStopped.Set(float64(stoppedCount))
	lm.metrics.XAppMetrics.InstancesFailed.Set(float64(failedCount))

	lm.logger.WithFields(logrus.Fields{
		"total_xapps":       len(lm.xapps),
		"total_instances":   len(lm.instances),
		"running_instances": runningCount,
		"stopped_instances": stoppedCount,
		"failed_instances":  failedCount,
	}).Debug("xApp metrics collected")
}

func (lm *LifecycleManager) instanceMonitor() {
	defer lm.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-ticker.C:
			lm.monitorInstances()
		}
	}
}

func (lm *LifecycleManager) monitorInstances() {
	lm.xappsMutex.RLock()
	instances := make([]*XAppInstance, 0, len(lm.instances))
	for _, instance := range lm.instances {
		instances = append(instances, instance)
	}
	lm.xappsMutex.RUnlock()

	for _, instance := range instances {
		// Skip non-running instances
		if instance.Status != XAppStatusRunning {
			continue
		}

		// Update runtime info and metrics
		ctx, cancel := context.WithTimeout(lm.ctx, 10*time.Second)

		// Get runtime info
		if runtimeInfo, err := lm.deploymentEngine.GetRuntimeInfo(ctx, instance.InstanceID); err == nil {
			lm.xappsMutex.Lock()
			if storedInstance, exists := lm.instances[instance.InstanceID]; exists {
				storedInstance.RuntimeInfo = *runtimeInfo
				storedInstance.UpdatedAt = time.Now()
			}
			lm.xappsMutex.Unlock()
		}

		// Get metrics
		if metrics, err := lm.deploymentEngine.GetMetrics(ctx, instance.InstanceID); err == nil {
			lm.xappsMutex.Lock()
			if storedInstance, exists := lm.instances[instance.InstanceID]; exists {
				storedInstance.Metrics = *metrics
				storedInstance.UpdatedAt = time.Now()
			}
			lm.xappsMutex.Unlock()
		}

		cancel()
	}
}

// Public interface methods

func (lm *LifecycleManager) AddEventHandler(handler XAppEventHandler) {
	lm.eventHandlers = append(lm.eventHandlers, handler)
}

func (lm *LifecycleManager) GetStats() map[string]interface{} {
	lm.xappsMutex.RLock()
	defer lm.xappsMutex.RUnlock()

	var runningCount, stoppedCount, failedCount int

	for _, instance := range lm.instances {
		switch instance.Status {
		case XAppStatusRunning:
			runningCount++
		case XAppStatusStopped:
			stoppedCount++
		case XAppStatusFailed:
			failedCount++
		}
	}

	return map[string]interface{}{
		"total_xapps":       len(lm.xapps),
		"total_instances":   len(lm.instances),
		"running_instances": runningCount,
		"stopped_instances": stoppedCount,
		"failed_instances":  failedCount,
		"max_instances":     lm.maxInstances,
	}
}
