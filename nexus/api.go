// Package nexus provides client and server implementations of the Nexus [HTTP API]
//
// [HTTP API]: https://github.com/nexus-rpc/api
package nexus

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"regexp"
)

// Package version.
const Version = "dev"

// TODO: ^^^ Actual version as part of the release tagging process.

const (
	headerContentType    = "Content-Type"
	headerOperationState = "Nexus-Operation-State"
	headerOperationID    = "Nexus-Operation-Id"
	headerRequestID      = "Nexus-Request-Id"
)

const contentTypeJSON = "application/json"

// Query param for passing a callback URL.
const queryCallbackURL = "callback"

// Query param for passing wait duration.
const queryWait = "wait"

const statusOperationRunning = http.StatusPreconditionFailed

// HTTP status code for failed operation responses.
const statusOperationFailed = http.StatusFailedDependency

// Failure represents protocol level failures returned in non successful HTTP responses as well as `failed` or
// `canceled` operation results.
type Failure struct {
	// A simple text message.
	Message string `json:"message"`
	// A key-value mapping for additional context. Useful for decoding the 'details' field, if needed.
	Metadata map[string]string `json:"metadata,omitempty"`
	// Additional JSON serializable structured data.
	Details json.RawMessage `json:"details,omitempty"`
}

// UnsuccessfulOperationError represents "failed" and "canceled" operation results.
type UnsuccessfulOperationError struct {
	State   OperationState
	Failure *Failure
}

// Error implements the error interface.
func (e *UnsuccessfulOperationError) Error() string {
	if e.Failure.Message != "" {
		return fmt.Sprintf("operation %s: %s", e.State, e.Failure.Message)
	}
	return fmt.Sprintf("operation %s", e.State)
}

// OperationInfo conveys information about an operation.
type OperationInfo struct {
	// ID of the operation.
	ID string `json:"id"`
	// State of the operation.
	State OperationState `json:"state"`
}

// OperationState represents the variable states of an operation.
type OperationState string

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

// isContentTypeJSON returns true if header contains a parsable Content-Type header with media type of application/json.
func isContentTypeJSON(header http.Header) bool {
	contentType := header.Get(headerContentType)
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == contentTypeJSON
}

var isValidOperationName = regexp.MustCompile(`^[\w\-.~]+$`)
var isValidOperationID = isValidOperationName // same rules apply
