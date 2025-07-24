package a1

import (
	"errors"
	"sync"
)

// PolicyManager manages the lifecycle of A1 policies.
type PolicyManager struct {
	mutex             sync.RWMutex
	policyTypes       map[string]PolicyType
	policyInstances   map[string]PolicyInstance
}

// NewPolicyManager creates a new PolicyManager.
func NewPolicyManager() *PolicyManager {
	return &PolicyManager{
		policyTypes:     make(map[string]PolicyType),
		policyInstances: make(map[string]PolicyInstance),
	}
}

// CreatePolicyType creates a new policy type.
func (pm *PolicyManager) CreatePolicyType(pt PolicyType) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.policyTypes[pt.ID]; exists {
		return errors.New("policy type already exists")
	}

	pm.policyTypes[pt.ID] = pt
	return nil
}

// GetPolicyType returns a policy type by its ID.
func (pm *PolicyManager) GetPolicyType(id string) (PolicyType, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pt, exists := pm.policyTypes[id]
	if !exists {
		return PolicyType{}, errors.New("policy type not found")
	}

	return pt, nil
}

// CreatePolicyInstance creates a new policy instance.
func (pm *PolicyManager) CreatePolicyInstance(pi PolicyInstance) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.policyInstances[pi.ID]; exists {
		return errors.New("policy instance already exists")
	}

	pm.policyInstances[pi.ID] = pi
	return nil
}

// GetPolicyInstance returns a policy instance by its ID.
func (pm *PolicyManager) GetPolicyInstance(id string) (PolicyInstance, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pi, exists := pm.policyInstances[id]
	if !exists {
		return PolicyInstance{}, errors.New("policy instance not found")
	}

	return pi, nil
}
