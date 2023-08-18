package nexusapi

import (
	"encoding/json"
	"fmt"
)

type (
	// Failure represents protocol level failures returned in non successful HTTP responses as well as `failed` or
	// `canceled` operation results.
	Failure struct {
		// A simple text message.
		Message string `json:"message"`
		// A key-value mapping for additional context. Useful for decoding the 'details' field, if needed.
		Metadata map[string]string `json:"metadata"`
		// Additional JSON serializable structured data.
		Details json.RawMessage `json:"details"`
	}

	UnsuccessfulOperationError struct {
		State   OperationState
		Failure Failure
	}
)

const (
	StatusOperationFailed = 482
)

func (e *UnsuccessfulOperationError) Error() string {
	if e.Failure.Message != "" {
		return fmt.Sprintf("operation %s: %s", e.State, e.Failure.Message)
	}
	return fmt.Sprintf("operation %s", e.State)
}
