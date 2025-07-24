package a1

import (
	"errors"
	"sync"
)

// MLModelManager manages the lifecycle of ML models.
type MLModelManager struct {
	mutex  sync.RWMutex
	models map[string]MLModel
}

// NewMLModelManager creates a new MLModelManager.
func NewMLModelManager() *MLModelManager {
	return &MLModelManager{
		models: make(map[string]MLModel),
	}
}

// CreateMLModel creates a new ML model.
func (mm *MLModelManager) CreateMLModel(m MLModel) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if _, exists := mm.models[m.ID]; exists {
		return errors.New("model already exists")
	}

	mm.models[m.ID] = m
	return nil
}

// GetMLModel returns an ML model by its ID.
func (mm *MLModelManager) GetMLModel(id string) (MLModel, error) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	m, exists := mm.models[id]
	if !exists {
		return MLModel{}, errors.New("model not found")
	}

	return m, nil
}