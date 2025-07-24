package xapp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// DependencyResolver resolves xApp dependencies
type DependencyResolver struct {
	lifecycleManager *LifecycleManager
	logger           *logrus.Entry

	// Dependency graph
	dependencyGraph map[XAppID][]XAppID

	// Service registry for dependency resolution
	serviceRegistry map[string]*ServiceEndpoint
}

// ServiceEndpoint represents a service endpoint for dependency resolution
type ServiceEndpoint struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // xapp, service, database
	Version     string                 `json:"version"`
	Endpoint    string                 `json:"endpoint"`
	Port        int32                  `json:"port"`
	Protocol    string                 `json:"protocol"`
	HealthCheck string                 `json:"health_check"`
	Metadata    map[string]interface{} `json:"metadata"`
	ProvidedBy  XAppID                 `json:"provided_by"`
	Available   bool                   `json:"available"`
}

// DependencyResolutionResult represents the result of dependency resolution
type DependencyResolutionResult struct {
	Success              bool                        `json:"success"`
	ResolvedServices     map[string]*ServiceEndpoint `json:"resolved_services"`
	MissingDependencies  []XAppDependency            `json:"missing_dependencies"`
	ConflictingServices  []string                    `json:"conflicting_services"`
	CircularDependencies [][]XAppID                  `json:"circular_dependencies"`
	ErrorMessage         string                      `json:"error_message"`
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(lm *LifecycleManager, baseLogger *logrus.Logger) *DependencyResolver {
	return &DependencyResolver{
		lifecycleManager: lm,
		logger:           baseLogger.WithField("component", "dependency-resolver"),
		dependencyGraph:  make(map[XAppID][]XAppID),
		serviceRegistry:  make(map[string]*ServiceEndpoint),
	}
}

// ResolveDependencies resolves dependencies for an xApp
func (dr *DependencyResolver) ResolveDependencies(xapp *XApp) error {
	dr.logger.WithFields(logrus.Fields{
		"xapp_id":            xapp.XAppID,
		"dependencies_count": len(xapp.Dependencies),
	}).Debug("Resolving xApp dependencies")

	result := dr.resolveDependenciesInternal(xapp)

	if !result.Success {
		return fmt.Errorf("dependency resolution failed: %s", result.ErrorMessage)
	}

	// Update dependency graph
	dr.updateDependencyGraph(xapp)

	dr.logger.WithField("xapp_id", xapp.XAppID).Info("Dependencies resolved successfully")
	return nil
}

// resolveDependenciesInternal performs the actual dependency resolution
func (dr *DependencyResolver) resolveDependenciesInternal(xapp *XApp) *DependencyResolutionResult {
	result := &DependencyResolutionResult{
		Success:              true,
		ResolvedServices:     make(map[string]*ServiceEndpoint),
		MissingDependencies:  []XAppDependency{},
		ConflictingServices:  []string{},
		CircularDependencies: [][]XAppID{},
	}

	// Check for circular dependencies first
	circularDeps := dr.detectCircularDependencies(xapp.XAppID, xapp.Dependencies)
	if len(circularDeps) > 0 {
		result.Success = false
		result.CircularDependencies = circularDeps
		result.ErrorMessage = fmt.Sprintf("Circular dependencies detected: %v", circularDeps)
		return result
	}

	// Resolve each dependency
	for _, dep := range xapp.Dependencies {
		resolved, err := dr.resolveSingleDependency(dep)
		if err != nil {
			if dep.Required {
				result.Success = false
				result.MissingDependencies = append(result.MissingDependencies, dep)
				if result.ErrorMessage == "" {
					result.ErrorMessage = err.Error()
				} else {
					result.ErrorMessage += "; " + err.Error()
				}
			} else {
				dr.logger.WithFields(logrus.Fields{
					"dependency": dep.Name,
					"error":      err.Error(),
				}).Warn("Optional dependency not resolved")
			}
			continue
		}

		// Check for conflicts
		if existing, exists := result.ResolvedServices[dep.Name]; exists {
			if existing.Version != resolved.Version {
				result.ConflictingServices = append(result.ConflictingServices, dep.Name)
				result.Success = false
				result.ErrorMessage = fmt.Sprintf("Version conflict for service %s: %s vs %s",
					dep.Name, existing.Version, resolved.Version)
			}
		} else {
			result.ResolvedServices[dep.Name] = resolved
		}
	}

	return result
}

// resolveSingleDependency resolves a single dependency
func (dr *DependencyResolver) resolveSingleDependency(dep XAppDependency) (*ServiceEndpoint, error) {
	dr.logger.WithFields(logrus.Fields{
		"dependency_name": dep.Name,
		"dependency_type": dep.Type,
		"version":         dep.Version,
		"required":        dep.Required,
	}).Debug("Resolving dependency")

	switch dep.Type {
	case "xapp":
		return dr.resolveXAppDependency(dep)
	case "service":
		return dr.resolveServiceDependency(dep)
	case "database":
		return dr.resolveDatabaseDependency(dep)
	default:
		return nil, fmt.Errorf("unsupported dependency type: %s", dep.Type)
	}
}

// resolveXAppDependency resolves an xApp dependency
func (dr *DependencyResolver) resolveXAppDependency(dep XAppDependency) (*ServiceEndpoint, error) {
	// Check if the required xApp is already registered
	xapps := dr.lifecycleManager.ListXApps()

	for _, xapp := range xapps {
		if xapp.Name == dep.Name {
			// Check version compatibility
			if dep.Version != "" && !dr.isVersionCompatible(xapp.Version, dep.Version) {
				continue
			}

			// Check if xApp has running instances
			instances := dr.lifecycleManager.GetInstancesByXApp(xapp.XAppID)
			var runningInstance *XAppInstance
			for _, instance := range instances {
				if instance.Status == XAppStatusRunning {
					runningInstance = instance
					break
				}
			}

			if runningInstance == nil {
				return nil, fmt.Errorf("xApp %s has no running instances", dep.Name)
			}

			// Create service endpoint from xApp interface
			endpoint := dr.createEndpointFromXApp(xapp, runningInstance, dep.Interface)
			if endpoint != nil {
				dr.registerService(endpoint)
				return endpoint, nil
			}
		}
	}

	return nil, fmt.Errorf("xApp dependency %s not found or not compatible", dep.Name)
}

// resolveServiceDependency resolves a service dependency
func (dr *DependencyResolver) resolveServiceDependency(dep XAppDependency) (*ServiceEndpoint, error) {
	// Check service registry
	if service, exists := dr.serviceRegistry[dep.Name]; exists {
		if dep.Version != "" && !dr.isVersionCompatible(service.Version, dep.Version) {
			return nil, fmt.Errorf("service %s version %s not compatible with required %s",
				dep.Name, service.Version, dep.Version)
		}

		if !service.Available {
			return nil, fmt.Errorf("service %s is not available", dep.Name)
		}

		return service, nil
	}

	// Check for well-known services
	if endpoint := dr.resolveWellKnownService(dep); endpoint != nil {
		dr.registerService(endpoint)
		return endpoint, nil
	}

	return nil, fmt.Errorf("service dependency %s not found", dep.Name)
}

// resolveDatabaseDependency resolves a database dependency
func (dr *DependencyResolver) resolveDatabaseDependency(dep XAppDependency) (*ServiceEndpoint, error) {
	// Check for database services in registry
	for _, service := range dr.serviceRegistry {
		if service.Type == "database" &&
			(service.Name == dep.Name || strings.Contains(service.Name, dep.Name)) {
			if dep.Version != "" && !dr.isVersionCompatible(service.Version, dep.Version) {
				continue
			}
			if service.Available {
				return service, nil
			}
		}
	}

	// Create default database endpoints for common databases
	if endpoint := dr.createDefaultDatabaseEndpoint(dep); endpoint != nil {
		dr.registerService(endpoint)
		return endpoint, nil
	}

	return nil, fmt.Errorf("database dependency %s not found", dep.Name)
}

// Helper methods

func (dr *DependencyResolver) detectCircularDependencies(xappID XAppID, dependencies []XAppDependency) [][]XAppID {
	var circular [][]XAppID
	visited := make(map[XAppID]bool)
	recStack := make(map[XAppID]bool)

	// Build dependency map for this check
	depMap := make(map[XAppID][]XAppID)
	for _, dep := range dependencies {
		if dep.Type == "xapp" {
			// Find xApp ID by name
			xapps := dr.lifecycleManager.ListXApps()
			for _, xapp := range xapps {
				if xapp.Name == dep.Name {
					depMap[xappID] = append(depMap[xappID], xapp.XAppID)
					break
				}
			}
		}
	}

	// Merge with existing dependency graph
	for k, v := range dr.dependencyGraph {
		if existing, exists := depMap[k]; exists {
			depMap[k] = append(existing, v...)
		} else {
			depMap[k] = v
		}
	}

	// Check for cycles using DFS
	if dr.hasCycleDFS(xappID, depMap, visited, recStack) {
		// Reconstruct the cycle path (simplified)
		cycle := []XAppID{xappID}
		if deps, exists := depMap[xappID]; exists && len(deps) > 0 {
			cycle = append(cycle, deps[0])
		}
		circular = append(circular, cycle)
	}

	return circular
}

func (dr *DependencyResolver) hasCycleDFS(xappID XAppID, depMap map[XAppID][]XAppID, visited, recStack map[XAppID]bool) bool {
	visited[xappID] = true
	recStack[xappID] = true

	if deps, exists := depMap[xappID]; exists {
		for _, dep := range deps {
			if !visited[dep] {
				if dr.hasCycleDFS(dep, depMap, visited, recStack) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}
	}

	recStack[xappID] = false
	return false
}

func (dr *DependencyResolver) updateDependencyGraph(xapp *XApp) {
	var deps []XAppID

	for _, dep := range xapp.Dependencies {
		if dep.Type == "xapp" {
			// Find xApp ID by name
			xapps := dr.lifecycleManager.ListXApps()
			for _, depXApp := range xapps {
				if depXApp.Name == dep.Name {
					deps = append(deps, depXApp.XAppID)
					break
				}
			}
		}
	}

	dr.dependencyGraph[xapp.XAppID] = deps
}

func (dr *DependencyResolver) createEndpointFromXApp(xapp *XApp, instance *XAppInstance, interfaceName string) *ServiceEndpoint {
	// Find the specified interface
	var targetInterface interface{}

	// Check REST interfaces
	for _, restIntf := range xapp.Interfaces.REST {
		if interfaceName == "" || restIntf.Name == interfaceName {
			targetInterface = restIntf
			break
		}
	}

	if targetInterface == nil {
		// Check custom interfaces
		for _, customIntf := range xapp.Interfaces.Custom {
			if interfaceName == "" || customIntf.Name == interfaceName {
				targetInterface = customIntf
				break
			}
		}
	}

	if targetInterface == nil {
		return nil
	}

	// Create endpoint based on interface type
	switch intf := targetInterface.(type) {
	case RESTInterface:
		return &ServiceEndpoint{
			Name:        intf.Name,
			Type:        "xapp",
			Version:     xapp.Version,
			Endpoint:    instance.RuntimeInfo.IPAddress,
			Port:        intf.Port,
			Protocol:    intf.Protocol,
			HealthCheck: fmt.Sprintf("%s://%s:%d%s/health", intf.Protocol, instance.RuntimeInfo.IPAddress, intf.Port, intf.Path),
			Metadata: map[string]interface{}{
				"xapp_id":     xapp.XAppID,
				"instance_id": instance.InstanceID,
				"path":        intf.Path,
			},
			ProvidedBy: xapp.XAppID,
			Available:  instance.Status == XAppStatusRunning,
		}
	case CustomInterface:
		return &ServiceEndpoint{
			Name:        intf.Name,
			Type:        "xapp",
			Version:     xapp.Version,
			Endpoint:    instance.RuntimeInfo.IPAddress,
			Port:        intf.Port,
			Protocol:    intf.Protocol,
			HealthCheck: "",
			Metadata: map[string]interface{}{
				"xapp_id":       xapp.XAppID,
				"instance_id":   instance.InstanceID,
				"configuration": intf.Configuration,
			},
			ProvidedBy: xapp.XAppID,
			Available:  instance.Status == XAppStatusRunning,
		}
	}

	return nil
}

func (dr *DependencyResolver) resolveWellKnownService(dep XAppDependency) *ServiceEndpoint {
	// Define well-known services
	wellKnownServices := map[string]*ServiceEndpoint{
		"message-router": {
			Name:        "message-router",
			Type:        "service",
			Version:     "1.0",
			Endpoint:    "message-router-service",
			Port:        3904,
			Protocol:    "http",
			HealthCheck: "http://message-router-service:3904/health",
			Available:   true,
		},
		"config-server": {
			Name:        "config-server",
			Type:        "service",
			Version:     "1.0",
			Endpoint:    "config-server-service",
			Port:        8080,
			Protocol:    "http",
			HealthCheck: "http://config-server-service:8080/health",
			Available:   true,
		},
		"subscription-manager": {
			Name:        "subscription-manager",
			Type:        "service",
			Version:     "1.0",
			Endpoint:    "subscription-manager-service",
			Port:        8088,
			Protocol:    "http",
			HealthCheck: "http://subscription-manager-service:8088/health",
			Available:   true,
		},
	}

	if service, exists := wellKnownServices[dep.Name]; exists {
		if dep.Version == "" || dr.isVersionCompatible(service.Version, dep.Version) {
			return service
		}
	}

	return nil
}

func (dr *DependencyResolver) createDefaultDatabaseEndpoint(dep XAppDependency) *ServiceEndpoint {
	// Define default database endpoints
	defaultDatabases := map[string]*ServiceEndpoint{
		"postgresql": {
			Name:        "postgresql",
			Type:        "database",
			Version:     "13.0",
			Endpoint:    "postgresql-service",
			Port:        5432,
			Protocol:    "tcp",
			HealthCheck: "",
			Available:   true,
		},
		"influxdb": {
			Name:        "influxdb",
			Type:        "database",
			Version:     "2.0",
			Endpoint:    "influxdb-service",
			Port:        8086,
			Protocol:    "http",
			HealthCheck: "http://influxdb-service:8086/health",
			Available:   true,
		},
		"redis": {
			Name:        "redis",
			Type:        "database",
			Version:     "6.0",
			Endpoint:    "redis-service",
			Port:        6379,
			Protocol:    "tcp",
			HealthCheck: "",
			Available:   true,
		},
	}

	if db, exists := defaultDatabases[dep.Name]; exists {
		if dep.Version == "" || dr.isVersionCompatible(db.Version, dep.Version) {
			return db
		}
	}

	return nil
}

func (dr *DependencyResolver) isVersionCompatible(available, required string) bool {
	// Simplified version compatibility check
	// In production, implement proper semantic version comparison
	if required == "" || available == "" {
		return true
	}

	// Exact match
	if available == required {
		return true
	}

	// Major version compatibility (e.g., 1.x is compatible with 1.y)
	availableParts := strings.Split(available, ".")
	requiredParts := strings.Split(required, ".")

	if len(availableParts) > 0 && len(requiredParts) > 0 {
		return availableParts[0] == requiredParts[0]
	}

	return false
}

// Service registry management

// RegisterService registers a service in the registry
func (dr *DependencyResolver) RegisterService(service *ServiceEndpoint) {
	dr.registerService(service)
}

func (dr *DependencyResolver) registerService(service *ServiceEndpoint) {
	dr.serviceRegistry[service.Name] = service

	dr.logger.WithFields(logrus.Fields{
		"service_name": service.Name,
		"service_type": service.Type,
		"endpoint":     service.Endpoint,
		"port":         service.Port,
		"available":    service.Available,
	}).Debug("Service registered")
}

// UnregisterService removes a service from the registry
func (dr *DependencyResolver) UnregisterService(serviceName string) {
	delete(dr.serviceRegistry, serviceName)

	dr.logger.WithField("service_name", serviceName).Debug("Service unregistered")
}

// GetAvailableServices returns all available services
func (dr *DependencyResolver) GetAvailableServices() []*ServiceEndpoint {
	var services []*ServiceEndpoint
	for _, service := range dr.serviceRegistry {
		if service.Available {
			services = append(services, service)
		}
	}

	// Sort by name for consistent ordering
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services
}

// GetDependencyGraph returns the current dependency graph
func (dr *DependencyResolver) GetDependencyGraph() map[XAppID][]XAppID {
	// Return a copy to prevent external modification
	graph := make(map[XAppID][]XAppID)
	for k, v := range dr.dependencyGraph {
		deps := make([]XAppID, len(v))
		copy(deps, v)
		graph[k] = deps
	}
	return graph
}

// ValidateDependencies validates that all dependencies for an xApp can be resolved
func (dr *DependencyResolver) ValidateDependencies(xapp *XApp) *DependencyResolutionResult {
	return dr.resolveDependenciesInternal(xapp)
}

// GetStats returns dependency resolver statistics
func (dr *DependencyResolver) GetStats() map[string]interface{} {
	availableCount := 0
	for _, service := range dr.serviceRegistry {
		if service.Available {
			availableCount++
		}
	}

	return map[string]interface{}{
		"total_services":     len(dr.serviceRegistry),
		"available_services": availableCount,
		"dependency_graph":   len(dr.dependencyGraph),
		"services":           dr.GetAvailableServices(),
	}
}
