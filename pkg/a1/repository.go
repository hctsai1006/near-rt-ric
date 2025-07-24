package a1

import (
	"fmt"
	"sync"
	"time"
)

// MemoryRepository provides an in-memory implementation of the A1Repository interface.
type MemoryRepository struct {
	mutex        sync.RWMutex
	policyTypes  map[string]*A1PolicyType
	policies     map[string]*A1Policy
	enforcements map[string]*PolicyEnforcement
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		policyTypes:  make(map[string]*A1PolicyType),
		policies:     make(map[string]*A1Policy),
		enforcements: make(map[string]*PolicyEnforcement),
	}
}

func (r *MemoryRepository) CreatePolicyType(policyType *A1PolicyType) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.policyTypes[policyType.PolicyTypeID]; exists {
		return fmt.Errorf("policy type %s already exists", policyType.PolicyTypeID)
	}

	r.policyTypes[policyType.PolicyTypeID] = policyType
	return nil
}

func (r *MemoryRepository) GetPolicyType(policyTypeID string) (*A1PolicyType, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	policyType, exists := r.policyTypes[policyTypeID]
	if !exists {
		return nil, fmt.Errorf("policy type %s not found", policyTypeID)
	}

	return policyType, nil
}

func (r *MemoryRepository) UpdatePolicyType(policyType *A1PolicyType) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.policyTypes[policyType.PolicyTypeID]; !exists {
		return fmt.Errorf("policy type %s not found", policyType.PolicyTypeID)
	}

	r.policyTypes[policyType.PolicyTypeID] = policyType
	return nil
}

func (r *MemoryRepository) DeletePolicyType(policyTypeID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.policyTypes[policyTypeID]; !exists {
		return fmt.Errorf("policy type %s not found", policyTypeID)
	}

	delete(r.policyTypes, policyTypeID)
	return nil
}

func (r *MemoryRepository) ListPolicyTypes() ([]*A1PolicyType, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var policyTypes []*A1PolicyType
	for _, pt := range r.policyTypes {
		policyTypes = append(policyTypes, pt)
	}

	return policyTypes, nil
}

func (r *MemoryRepository) CreatePolicy(policy *A1Policy) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.policies[policy.PolicyID]; exists {
		return fmt.Errorf("policy %s already exists", policy.PolicyID)
	}

	r.policies[policy.PolicyID] = policy
	return nil
}

func (r *MemoryRepository) GetPolicy(policyID string) (*A1Policy, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	policy, exists := r.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy %s not found", policyID)
	}

	return policy, nil
}

func (r *MemoryRepository) UpdatePolicy(policy *A1Policy) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.policies[policy.PolicyID]; !exists {
		return fmt.Errorf("policy %s not found", policy.PolicyID)
	}

	r.policies[policy.PolicyID] = policy
	return nil
}

func (r *MemoryRepository) DeletePolicy(policyID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.policies[policyID]; !exists {
		return fmt.Errorf("policy %s not found", policyID)
	}

	delete(r.policies, policyID)
	return nil
}

func (r *MemoryRepository) ListPolicies() ([]*A1Policy, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var policies []*A1Policy
	for _, p := range r.policies {
		policies = append(policies, p)
	}

	return policies, nil
}

func (r *MemoryRepository) ListPoliciesByType(policyTypeID string) ([]*A1Policy, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var policies []*A1Policy
	for _, p := range r.policies {
		if p.PolicyTypeID == policyTypeID {
			policies = append(policies, p)
		}
	}

	return policies, nil
}

func (r *MemoryRepository) RecordEnforcement(enforcement *PolicyEnforcement) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.enforcements[enforcement.PolicyID] = enforcement
	return nil
}

func (r *MemoryRepository) GetEnforcementStatus(policyID string) (*PolicyEnforcement, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	enforcement, exists := r.enforcements[policyID]
	if !exists {
		return nil, fmt.Errorf("enforcement status for policy %s not found", policyID)
	}

	return enforcement, nil
}

func (r *MemoryRepository) GetPolicyMetrics(policyID string) (*PolicyMetrics, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	enforcement, exists := r.enforcements[policyID]
	if !exists {
		return nil, fmt.Errorf("enforcement status for policy %s not found", policyID)
	}

	return enforcement.Metrics, nil
}

func (r *MemoryRepository) GetPolicyTypeStatus(policyTypeID string) (*A1PolicyTypeStatus, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var instanceIDs []string
	for _, p := range r.policies {
		if p.PolicyTypeID == policyTypeID {
			instanceIDs = append(instanceIDs, p.PolicyID)
		}
	}

	return &A1PolicyTypeStatus{
		PolicyTypeID:      policyTypeID,
		NumberOfPolicies:  len(instanceIDs),
		PolicyInstanceIDs: instanceIDs,
	}, nil
}

func (r *MemoryRepository) GetStatistics() (map[string]interface{}, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["policy_types"] = len(r.policyTypes)
	stats["policies"] = len(r.policies)
	stats["enforcements"] = len(r.enforcements)

	return stats, nil
}

func (r *MemoryRepository) Export() ([]byte, error) {
	// Not implemented for in-memory repository
	return nil, fmt.Errorf("export not implemented for in-memory repository")
}

func (r *MemoryRepository) Import(data []byte) error {
	// Not implemented for in-memory repository
	return fmt.Errorf("import not implemented for in-memory repository")
}

func (r *MemoryRepository) Cleanup(maxAge time.Duration) error {
	// Not implemented for in-memory repository
	return nil
}
