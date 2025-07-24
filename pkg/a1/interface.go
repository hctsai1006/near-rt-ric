package a1

// A1Interface defines the A1 interface.
type A1Interface struct {
	PolicyManager     *PolicyManager
	MLModelManager    *MLModelManager
	EnrichmentManager *EnrichmentManager
}

// NewA1Interface creates a new A1Interface.
func NewA1Interface() *A1Interface {
	return &A1Interface{
		PolicyManager:     NewPolicyManager(),
		MLModelManager:    NewMLModelManager(),
		EnrichmentManager: NewEnrichmentManager(),
	}
}
