package nexustypes

import (
	"encoding/json"
)

type (
	// Payload is a generic data container.
	Payload struct {
		// A key-value mapping for additional context. Useful for decoding the 'data' field, if needed.
		Metadata map[string]string `json:"metadata"`
		// Arbitrary data.
		Data []byte `json:"data"`
	}

	// Failure represents protocol level failures returned in non successful HTTP responses as well as `failed` or
	// `canceled` operation results.
	Failure struct {
		// A simple text message.
		Message string `json:"message"`
		// Additional structured data. If this is byte data, it must be JSON serializable.
		Details any `json:"details"`
	}

	// PayloadFailure is a variant of the Failure struct, useful for transferring encrypted or binary information.
	// The Message field must be a Payload that can be decoded as string.
	PayloadFailure struct {
		Message Payload `json:"message"`
		Details Payload `json:"details"`
	}

	// RawFailure is a variant of the Failure struct that contains raw (unparsed) details.
	RawFailure struct {
		Message string          `json:"message"`
		Details json.RawMessage `json:"details"`
	}

	OperationState string

	OperationInfo struct {
		ID    string         `json:"id"`
		State OperationState `json:"state"`
	}
)

const (
	OperationStateRunning   = OperationState("running")
	OperationStateSucceeded = OperationState("succeeded")
	OperationStateFailed    = OperationState("failed")
	OperationStateCanceled  = OperationState("canceled")
)
