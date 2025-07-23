package yang

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/sirupsen/logrus"
)

// Manager handles YANG models
type Manager struct {
	modules map[string]*yang.Module
	logger  *logrus.Logger
}

// NewManager creates a new YANG manager
func NewManager(logger *logrus.Logger) *Manager {
	return &Manager{
		modules: make(map[string]*yang.Module),
		logger:  logger,
	}
}

// LoadModels loads all YANG models from a directory
func (m *Manager) LoadModels(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read YANG directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yang" {
			path := filepath.Join(dir, file.Name())
			m.logger.WithField("path", path).Info("Loading YANG model")
			module, err := yang.ParseFile(path, nil)
			if err != nil {
				m.logger.WithError(err).WithField("path", path).Error("Failed to parse YANG model")
				continue
			}
			m.modules[module.Name] = module
		}
	}

	m.logger.WithField("count", len(m.modules)).Info("YANG models loaded")
	return nil
}

// GetModule retrieves a YANG module by name
func (m *Manager) GetModule(name string) (*yang.Module, bool) {
	module, ok := m.modules[name]
	return module, ok
}
