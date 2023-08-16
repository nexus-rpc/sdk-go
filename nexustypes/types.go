package nexustypes

import (
	"encoding/json"
)

type (
	Payload struct {
		Metadata map[string]string `json:"metadata"`
		Data     []byte            `json:"data"`
	}

	// TODO: Find a better name for this
	UntypedFailure struct {
		Message any `json:"message"`
		Details any `json:"details"`
	}

	Failure struct {
		Message json.RawMessage `json:"message"`
		Details json.RawMessage `json:"details"`
	}

	OperationState string

	OperationInfo struct {
		ID    string         `json:"id"`
		State OperationState `json:"state"`
	}
)

const (
	contentTransferEncoding      = "Content-Transfer-Encoding"
	contentTransferEncodingLower = "content-transfer-encoding"
	OperationStateRunning        = OperationState("running")
	OperationStateSucceeded      = OperationState("succeeded")
	OperationStateFailed         = OperationState("failed")
	OperationStateCanceled       = OperationState("canceled")
)
