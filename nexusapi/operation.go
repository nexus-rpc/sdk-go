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
	// "running" operation state. Indicates an operation is started and not yet completed.
	OperationStateRunning OperationState = "running"
	// "succeeded" operation state. Indicates an operation completed successfully.
	OperationStateSucceeded OperationState = "succeeded"
	// "failed" operation state. Indicates an operation completed as failed.
	OperationStateFailed OperationState = "failed"
	// "canceled" operation state. Indicates an operation completed as canceled.
	OperationStateCanceled OperationState = "canceled"
)
