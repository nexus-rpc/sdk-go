package nexusapi

type (
	// OperationState represents the variable states of an operation.
	OperationState string

	// OperationInfo conveys information about an operation.
	OperationInfo struct {
		// ID of the operation.
		ID string `json:"id"`
		// State of the operation.
		State OperationState `json:"state"`
	}
)

const (
	OperationStateRunning   = OperationState("running")
	OperationStateSucceeded = OperationState("succeeded")
	OperationStateFailed    = OperationState("failed")
	OperationStateCanceled  = OperationState("canceled")
)
