package config

import (
	"sync"

	"github.com/hctsai1006/near-rt-ric/pkg/o1/yang"
	"github.com/sirupsen/logrus"
)

// ConfigurationManager handles configuration management
type ConfigurationManager struct {
	yangManager *yang.Manager
	logger      *logrus.Logger
	config      map[string]interface{}
	mutex       sync.RWMutex
}

// NewConfigurationManager creates a new configuration manager
func NewConfigurationManager(yangManager *yang.Manager, logger *logrus.Logger) *ConfigurationManager {
	return &ConfigurationManager{
		yangManager: yangManager,
		logger:      logger,
		config:      make(map[string]interface{}),
	}
}

// GetConfig retrieves the current configuration
func (m *ConfigurationManager) GetConfig() (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config, nil
}

// EditConfig modifies the configuration
func (m *ConfigurationManager) EditConfig(newConfig map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config = newConfig
	m.logger.Info("Configuration updated")
	return nil
}
