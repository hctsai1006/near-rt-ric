package xapp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// XAppInterface represents the main xApp framework interface
// Complies with O-RAN.WG2.xApp-v03.00 specification
type XAppInterface struct {
	logger  *logrus.Entry
	metrics *monitoring.MetricsCollector
	config  *config.XAppConfig

	// Core components
	lifecycleManager   *LifecycleManager
	healthMonitor      *HealthMonitor
	dependencyResolver *DependencyResolver
	deploymentEngine   DeploymentEngine

	// Control and synchronization
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mutex   sync.RWMutex

	// Event handlers
	eventHandlers []XAppInterfaceEventHandler

	// Statistics
	startTime time.Time
}

// XAppInterfaceEventHandler defines interface for xApp framework events
type XAppInterfaceEventHandler interface {
	OnXAppFrameworkStarted()
	OnXAppFrameworkStopped()
	OnXAppRegistered(xapp *XApp)
	OnXAppUnregistered(xappID XAppID)
	OnInstanceDeployed(instance *XAppInstance)
	OnInstanceFailed(instance *XAppInstance, err error)
	OnDependencyResolutionFailed(xappID XAppID, err error)
	OnHealthCheckFailed(instanceID XAppInstanceID, err error)
	OnError(err error)
}

// NewXAppInterface creates a new xApp framework interface
func NewXAppInterface(cfg *config.XAppConfig, baseLogger *logrus.Logger, metrics *monitoring.MetricsCollector) (*XAppInterface, error) {
	ctx, cancel := context.WithCancel(context.Background())

	xappIntf := &XAppInterface{
		logger:    baseLogger.WithField("component", "xapp-interface"),
		metrics:   metrics,
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}

	// Initialize deployment engine based on configuration
	var deploymentEngine DeploymentEngine
	switch cfg.Manager.DeploymentEngine {
	case "kubernetes":
		deploymentEngine = NewKubernetesDeploymentEngine(baseLogger, cfg.Kubernetes.Namespace)
	case "docker":
		deploymentEngine = NewDockerDeploymentEngine(baseLogger)
	case "mock":
		deploymentEngine = NewMockDeploymentEngine(baseLogger)
	default:
		cancel()
		return nil, fmt.Errorf("unsupported deployment engine: %s", cfg.Manager.DeploymentEngine)
	}

	xappIntf.deploymentEngine = deploymentEngine

	// Initialize lifecycle manager
	xappIntf.lifecycleManager = NewLifecycleManager(baseLogger, metrics, deploymentEngine)

	// Initialize health monitor
	xappIntf.healthMonitor = NewHealthMonitor(xappIntf.lifecycleManager, baseLogger, metrics)

	// Initialize dependency resolver
	xappIntf.dependencyResolver = NewDependencyResolver(xappIntf.lifecycleManager, baseLogger)

	// Set up event handlers for integrated components
	xappIntf.setupEventHandlers()

	xappIntf.logger.Info("xApp framework interface initialized successfully")
	return xappIntf, nil
}

// Start starts the xApp framework interface
func (xi *XAppInterface) Start(ctx context.Context) error {
	xi.mutex.Lock()
	defer xi.mutex.Unlock()

	if xi.running {
		return fmt.Errorf("xApp framework interface is already running")
	}

	xi.logger.Info("Starting O-RAN xApp framework interface")

	// Start lifecycle manager
	if err := xi.lifecycleManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start lifecycle manager: %w", err)
	}

	// Start health monitor
	if err := xi.healthMonitor.Start(ctx); err != nil {
		_ = xi.lifecycleManager.Stop(ctx)
		return fmt.Errorf("failed to start health monitor: %w", err)
	}

	// Start background monitoring
	xi.wg.Add(1)
	go xi.monitoringWorker()

	xi.running = true
	xi.logger.Info("O-RAN xApp framework interface started successfully")

	// Send event
	for _, handler := range xi.eventHandlers {
		go handler.OnXAppFrameworkStarted()
	}

	return nil
}

// Stop stops the xApp framework interface gracefully
func (xi *XAppInterface) Stop(ctx context.Context) error {
	xi.mutex.Lock()
	defer xi.mutex.Unlock()

	if !xi.running {
		return nil
	}

	xi.logger.Info("Stopping O-RAN xApp framework interface")

	// Stop health monitor
	if err := xi.healthMonitor.Stop(ctx); err != nil {
		xi.logger.WithError(err).Error("Error stopping health monitor")
	}

	// Stop lifecycle manager
	if err := xi.lifecycleManager.Stop(ctx); err != nil {
		xi.logger.WithError(err).Error("Error stopping lifecycle manager")
	}

	// Cancel context and wait for workers
	xi.cancel()

	done := make(chan struct{})
	go func() {
		xi.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		xi.logger.Info("O-RAN xApp framework interface stopped successfully")
	case <-ctx.Done():
		xi.logger.Warn("xApp framework interface shutdown timeout")
	}

	xi.running = false

	// Send event
	for _, handler := range xi.eventHandlers {
		go handler.OnXAppFrameworkStopped()
	}

	return nil
}

// XApp Management Methods

// RegisterXApp registers a new xApp in the framework
func (xi *XAppInterface) RegisterXApp(xapp *XApp) error {
	xi.logger.WithFields(logrus.Fields{
		"xapp_id": xapp.XAppID,
		"name":    xapp.Name,
		"version": xapp.Version,
		"type":    xapp.Type,
	}).Info("Registering xApp")

	// Validate dependencies first
	result := xi.dependencyResolver.ValidateDependencies(xapp)
	if !result.Success {
		err := fmt.Errorf("dependency validation failed: %s", result.ErrorMessage)

		// Send event
		for _, handler := range xi.eventHandlers {
			go handler.OnDependencyResolutionFailed(xapp.XAppID, err)
		}

		return err
	}

	// Register with lifecycle manager
	if err := xi.lifecycleManager.RegisterXApp(xapp); err != nil {
		return fmt.Errorf("failed to register xApp: %w", err)
	}

	// Send event
	for _, handler := range xi.eventHandlers {
		go handler.OnXAppRegistered(xapp)
	}

	xi.logger.WithField("xapp_id", xapp.XAppID).Info("xApp registered successfully")
	return nil
}

// UnregisterXApp removes an xApp from the framework
func (xi *XAppInterface) UnregisterXApp(xappID XAppID) error {
	xi.logger.WithField("xapp_id", xappID).Info("Unregistering xApp")

	if err := xi.lifecycleManager.UnregisterXApp(xappID); err != nil {
		return fmt.Errorf("failed to unregister xApp: %w", err)
	}

	// Send event
	for _, handler := range xi.eventHandlers {
		go handler.OnXAppUnregistered(xappID)
	}

	xi.logger.WithField("xapp_id", xappID).Info("xApp unregistered successfully")
	return nil
}

// DeployXAppInstance deploys a new instance of an xApp
func (xi *XAppInterface) DeployXAppInstance(xappID XAppID, config map[string]interface{}) (*XAppInstance, error) {
	xi.logger.WithField("xapp_id", xappID).Info("Deploying xApp instance")

	instance, err := xi.lifecycleManager.DeployInstance(xappID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy xApp instance: %w", err)
	}

	xi.logger.WithFields(logrus.Fields{
		"xapp_id":     xappID,
		"instance_id": instance.InstanceID,
	}).Info("xApp instance deployment initiated")

	return instance, nil
}

// StopXAppInstance stops a running xApp instance
func (xi *XAppInterface) StopXAppInstance(instanceID XAppInstanceID) error {
	return xi.lifecycleManager.StopInstance(instanceID)
}

// DeleteXAppInstance deletes an xApp instance
func (xi *XAppInterface) DeleteXAppInstance(instanceID XAppInstanceID) error {
	return xi.lifecycleManager.DeleteInstance(instanceID)
}

// UpdateXAppInstanceConfig updates the configuration of a running instance
func (xi *XAppInterface) UpdateXAppInstanceConfig(instanceID XAppInstanceID, config map[string]interface{}) error {
	return xi.lifecycleManager.UpdateInstanceConfig(instanceID, config)
}

// Query Methods

// GetXApp retrieves an xApp by ID
func (xi *XAppInterface) GetXApp(xappID XAppID) (*XApp, error) {
	return xi.lifecycleManager.GetXApp(xappID)
}

// ListXApps returns all registered xApps
func (xi *XAppInterface) ListXApps() []*XApp {
	return xi.lifecycleManager.ListXApps()
}

// GetXAppInstance retrieves an instance by ID
func (xi *XAppInterface) GetXAppInstance(instanceID XAppInstanceID) (*XAppInstance, error) {
	return xi.lifecycleManager.GetInstance(instanceID)
}

// ListXAppInstances returns all instances
func (xi *XAppInterface) ListXAppInstances() []*XAppInstance {
	return xi.lifecycleManager.ListInstances()
}

// GetXAppInstancesByXApp returns instances for a specific xApp
func (xi *XAppInterface) GetXAppInstancesByXApp(xappID XAppID) []*XAppInstance {
	return xi.lifecycleManager.GetInstancesByXApp(xappID)
}

// Health and Monitoring Methods

// PerformHealthCheck performs a health check on a specific instance
func (xi *XAppInterface) PerformHealthCheck(instanceID XAppInstanceID) (*HealthCheckResult, error) {
	return xi.healthMonitor.PerformHealthCheck(instanceID)
}

// GetHealthHistory returns health check history for an instance
func (xi *XAppInterface) GetHealthHistory(instanceID XAppInstanceID, limit int) []HealthCheckResult {
	return xi.healthMonitor.GetHealthHistory(instanceID, limit)
}

// GetOverallHealth returns overall health status for all instances
func (xi *XAppInterface) GetOverallHealth() map[string]interface{} {
	return xi.healthMonitor.GetOverallHealth()
}

// Dependency Management Methods

// ValidateXAppDependencies validates dependencies for an xApp
func (xi *XAppInterface) ValidateXAppDependencies(xapp *XApp) *DependencyResolutionResult {
	return xi.dependencyResolver.ValidateDependencies(xapp)
}

// GetAvailableServices returns all available services for dependency resolution
func (xi *XAppInterface) GetAvailableServices() []*ServiceEndpoint {
	return xi.dependencyResolver.GetAvailableServices()
}

// RegisterService registers a service endpoint for dependency resolution
func (xi *XAppInterface) RegisterService(service *ServiceEndpoint) {
	xi.dependencyResolver.RegisterService(service)
}

// UnregisterService removes a service from the registry
func (xi *XAppInterface) UnregisterService(serviceName string) {
	xi.dependencyResolver.UnregisterService(serviceName)
}

// GetDependencyGraph returns the current dependency graph
func (xi *XAppInterface) GetDependencyGraph() map[XAppID][]XAppID {
	return xi.dependencyResolver.GetDependencyGraph()
}

// Background workers

func (xi *XAppInterface) monitoringWorker() {
	defer xi.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-xi.ctx.Done():
			return
		case <-ticker.C:
			xi.collectFrameworkMetrics()
		}
	}
}

func (xi *XAppInterface) collectFrameworkMetrics() {
	// Collect framework-level metrics
	stats := xi.GetStats()

	// Update Prometheus metrics
	if totalXApps, ok := stats["total_xapps"].(int); ok {
		xi.metrics.XAppMetrics.XAppsTotal.WithLabelValues("all").Set(float64(totalXApps))
	}

	if totalInstances, ok := stats["total_instances"].(int); ok {
		xi.metrics.XAppMetrics.InstancesTotal.Set(float64(totalInstances))
	}

	// Log framework statistics
	xi.logger.WithFields(logrus.Fields{
		"total_xapps":       stats["total_xapps"],
		"total_instances":   stats["total_instances"],
		"running_instances": stats["running_instances"],
		"failed_instances":  stats["failed_instances"],
		"uptime":            time.Since(xi.startTime),
	}).Debug("xApp framework statistics")
}

// Event handler setup

func (xi *XAppInterface) setupEventHandlers() {
	// Set up lifecycle manager event handler
	xi.lifecycleManager.AddEventHandler(&xappInterfaceEventHandler{xi})

	// Set up health monitor event handler
	xi.healthMonitor.AddEventHandler(&xappInterfaceHealthEventHandler{xi})
}

// Event handler implementations

type xappInterfaceEventHandler struct {
	xi *XAppInterface
}

func (h *xappInterfaceEventHandler) OnXAppCreated(xapp *XApp)    {}
func (h *xappInterfaceEventHandler) OnXAppUpdated(xapp *XApp)    {}
func (h *xappInterfaceEventHandler) OnXAppDeleted(xappID XAppID) {}

func (h *xappInterfaceEventHandler) OnInstanceCreated(instance *XAppInstance) {}

func (h *xappInterfaceEventHandler) OnInstanceStarted(instance *XAppInstance) {
	for _, handler := range h.xi.eventHandlers {
		go handler.OnInstanceDeployed(instance)
	}
}

func (h *xappInterfaceEventHandler) OnInstanceStopped(instance *XAppInstance) {}

func (h *xappInterfaceEventHandler) OnInstanceFailed(instance *XAppInstance, err error) {
	for _, handler := range h.xi.eventHandlers {
		go handler.OnInstanceFailed(instance, err)
	}
}

func (h *xappInterfaceEventHandler) OnInstanceDeleted(instanceID XAppInstanceID) {}

func (h *xappInterfaceEventHandler) OnHealthChanged(instanceID XAppInstanceID, oldState, newState HealthState) {
}

type xappInterfaceHealthEventHandler struct {
	xi *XAppInterface
}

func (h *xappInterfaceHealthEventHandler) OnHealthCheckStarted(instanceID XAppInstanceID) {}
func (h *xappInterfaceHealthEventHandler) OnHealthCheckCompleted(instanceID XAppInstanceID, result *HealthCheckResult) {
}
func (h *xappInterfaceHealthEventHandler) OnHealthStateChanged(instanceID XAppInstanceID, oldState, newState HealthState) {
}

func (h *xappInterfaceHealthEventHandler) OnHealthCheckFailed(instanceID XAppInstanceID, err error) {
	for _, handler := range h.xi.eventHandlers {
		go handler.OnHealthCheckFailed(instanceID, err)
	}
}

// Public interface methods

// IsRunning returns whether the xApp framework interface is running
func (xi *XAppInterface) IsRunning() bool {
	xi.mutex.RLock()
	defer xi.mutex.RUnlock()
	return xi.running
}

// AddEventHandler adds an event handler
func (xi *XAppInterface) AddEventHandler(handler XAppInterfaceEventHandler) {
	xi.eventHandlers = append(xi.eventHandlers, handler)
}

// GetStats returns xApp framework statistics
func (xi *XAppInterface) GetStats() map[string]interface{} {
	lifecycleStats := xi.lifecycleManager.GetStats()
	healthStats := xi.healthMonitor.GetHealthStats()
	dependencyStats := xi.dependencyResolver.GetStats()

	return map[string]interface{}{
		"running":           xi.IsRunning(),
		"uptime":            time.Since(xi.startTime),
		"deployment_engine": xi.config.Manager.DeploymentEngine,
		"lifecycle_stats":   lifecycleStats,
		"health_stats":      healthStats,
		"dependency_stats":  dependencyStats,
		"total_xapps":       lifecycleStats["total_xapps"],
		"total_instances":   lifecycleStats["total_instances"],
		"running_instances": lifecycleStats["running_instances"],
		"stopped_instances": lifecycleStats["stopped_instances"],
		"failed_instances":  lifecycleStats["failed_instances"],
	}
}

// HealthCheck performs a health check of the xApp framework interface
func (xi *XAppInterface) HealthCheck() error {
	if !xi.IsRunning() {
		return fmt.Errorf("xApp framework interface is not running")
	}

	// Check if critical components are healthy
	overallHealth := xi.GetOverallHealth()
	if status, ok := overallHealth["overall_status"].(string); ok {
		if status == "unhealthy" {
			return fmt.Errorf("xApp framework has unhealthy instances")
		}
	}

	return nil
}
