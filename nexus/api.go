// Package nexus provides client and server implementations of the Nexus [HTTP API]
//
// [HTTP API]: https://github.com/nexus-rpc/api
package nexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
)

// Package version.
// TODO: Actual version as part of the release tagging process.
const version = "dev"

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
	Failure Failure
}

// Error implements the error interface.
func (e *UnsuccessfulOperationError) Error() string {
	if e.Failure.Message != "" {
		return fmt.Sprintf("operation %s: %s", e.State, e.Failure.Message)
	}
	return fmt.Sprintf("operation %s", e.State)
}

// ErrOperationStillRunning indicates that an operation is still running while trying to get its result.
var ErrOperationStillRunning = errors.New("operation still running")

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
	return isMediaTypeJSON(header.Get(headerContentType))
}

// isMediaTypeJSON returns true if the given content type's media type is application/json.
func isMediaTypeJSON(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

// isMediaTypeOctetStream returns true if the given content type's media type is application/octet-stream.
func isMediaTypeOctetStream(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/octet-stream"
}

// Header is a mapping of string to string.
// It is used throughout the framework to transmit metadata.
type Header map[string]string

// Get is a case-insensitive key lookup from the header map.
func (h Header) Get(k string) string {
	return h[strings.ToLower(k)]
}

func httpHeaderToContentHeader(httpHeader http.Header) Header {
	header := Header{}
	for k, v := range httpHeader {
		if strings.HasPrefix(k, "Content-") {
			// Nexus headers can only have single values, ignore multiple values.
			header[strings.ToLower(k[8:])] = v[0]
		}
	}
	return header
}

func addContentHeaderToHTTPHeader(nexusHeader Header, httpHeader http.Header) {
	for k, v := range nexusHeader {
		// Nexus headers can only have single values, ignore multiple values.
		httpHeader.Set("Content-"+k, v)
	}
}

func httpHeaderToNexusHeader(httpHeader http.Header) Header {
	header := Header{}
	for k, v := range httpHeader {
		if !strings.HasPrefix(k, "Content-") {
			// Nexus headers can only have single values, ignore multiple values.
			header[strings.ToLower(k)] = v[0]
		}
	}
	return header
}

func addNexusHeaderToHTTPHeader(nexusHeader Header, httpHeader http.Header) {
	for k, v := range nexusHeader {
		httpHeader.Set(k, v)
	}
}
