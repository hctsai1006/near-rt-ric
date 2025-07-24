package e2

// E2APProcedureCode is a type for E2AP procedure codes.
type E2APProcedureCode int64

const (
	// E2APProcedureCodeSetup is the procedure code for the E2 Setup procedure.
	E2APProcedureCodeSetup E2APProcedureCode = 1
	// E2APProcedureCodeSubscription is the procedure code for the RIC Subscription procedure.
	E2APProcedureCodeSubscription E2APProcedureCode = 12
	// E2APProcedureCodeControl is the procedure code for the RIC Control procedure.
	E2APProcedureCodeControl E2APProcedureCode = 13
)