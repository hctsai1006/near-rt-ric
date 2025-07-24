package a1

import (
	"errors"
	"sync"
)

// EnrichmentManager manages the lifecycle of enrichment information.
type EnrichmentManager struct {
	mutex sync.RWMutex
	info  map[string]EnrichmentInfo
}

// NewEnrichmentManager creates a new EnrichmentManager.
func NewEnrichmentManager() *EnrichmentManager {
	return &EnrichmentManager{
		info: make(map[string]EnrichmentInfo),
	}
}

// CreateEnrichmentInfo creates a new enrichment information.
func (em *EnrichmentManager) CreateEnrichmentInfo(ei EnrichmentInfo) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if _, exists := em.info[ei.ID]; exists {
		return errors.New("enrichment info already exists")
	}

	em.info[ei.ID] = ei
	return nil
}

// GetEnrichmentInfo returns an enrichment information by its ID.
func (em *EnrichmentManager) GetEnrichmentInfo(id string) (EnrichmentInfo, error) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	ei, exists := em.info[id]
	if !exists {
		return EnrichmentInfo{}, errors.New("enrichment info not found")
	}

	return ei, nil
}