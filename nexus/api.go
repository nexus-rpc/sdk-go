// Package nexus provides client and server implementations of the Nexus [HTTP API]
//
// [HTTP API]: https://github.com/nexus-rpc/api
package nexus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"
)

// Package version.
const version = "v0.0.9"

const (
	// Nexus specific headers.
	headerOperationState = "Nexus-Operation-State"
	headerOperationID    = "Nexus-Operation-Id"
	headerRequestID      = "Nexus-Request-Id"
	headerLinks          = "Nexus-Link"

	// HeaderRequestTimeout is the total time to complete a Nexus HTTP request.
	HeaderRequestTimeout = "Request-Timeout"
)

const contentTypeJSON = "application/json"

// Query param for passing a callback URL.
const (
	queryCallbackURL = "callback"
	// Query param for passing wait duration.
	queryWait = "wait"
)

const (
	statusOperationRunning = http.StatusPreconditionFailed
	// HTTP status code for failed operation responses.
	statusOperationFailed   = http.StatusFailedDependency
	StatusDownstreamTimeout = 520
)

// A Failure represents failed handler invocations as well as `failed` or `canceled` operation results.
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

func prefixStrippedHTTPHeaderToNexusHeader(httpHeader http.Header, prefix string) Header {
	header := Header{}
	for k, v := range httpHeader {
		lowerK := strings.ToLower(k)
		if strings.HasPrefix(lowerK, prefix) {
			// Nexus headers can only have single values, ignore multiple values.
			header[lowerK[len(prefix):]] = v[0]
		}
	}
	return header
}

func addContentHeaderToHTTPHeader(nexusHeader Header, httpHeader http.Header) http.Header {
	for k, v := range nexusHeader {
		httpHeader.Set("Content-"+k, v)
	}
	return httpHeader
}

func addCallbackHeaderToHTTPHeader(nexusHeader Header, httpHeader http.Header) http.Header {
	for k, v := range nexusHeader {
		httpHeader.Set("Nexus-Callback-"+k, v)
	}
	return httpHeader
}

func addLinksToHTTPHeader(links []Link, httpHeader http.Header) error {
	for _, link := range links {
		encodedLink, err := encodeLink(link)
		if err != nil {
			return err
		}
		httpHeader.Add(headerLinks, encodedLink)
	}
	return nil
}

func getLinksFromHeader(httpHeader http.Header) ([]Link, error) {
	var links []Link
	for _, encodedLink := range httpHeader.Values(headerLinks) {
		link, err := decodeLink(encodedLink)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func httpHeaderToNexusHeader(httpHeader http.Header, excludePrefixes ...string) Header {
	header := Header{}
headerLoop:
	for k, v := range httpHeader {
		lowerK := strings.ToLower(k)
		for _, prefix := range excludePrefixes {
			if strings.HasPrefix(lowerK, prefix) {
				continue headerLoop
			}
		}
		// Nexus headers can only have single values, ignore multiple values.
		header[lowerK] = v[0]
	}
	return header
}

func addNexusHeaderToHTTPHeader(nexusHeader Header, httpHeader http.Header) http.Header {
	for k, v := range nexusHeader {
		httpHeader.Set(k, v)
	}
	return httpHeader
}

func addContextTimeoutToHTTPHeader(ctx context.Context, httpHeader http.Header) http.Header {
	deadline, ok := ctx.Deadline()
	if !ok {
		return httpHeader
	}
	httpHeader.Set(HeaderRequestTimeout, time.Until(deadline).String())
	return httpHeader
}

type Link struct {
	// Encoded representation of arbitrary link information.
	Data []byte `json:"data"`
	// Type of the `Data` field for encoding/decoding it.
	Type string `json:"type"`
}

var linkBase64Encoding = base64.StdEncoding

func encodeLink(link Link) (string, error) {
	linkJson, err := json.Marshal(link)
	if err != nil {
		return "", err
	}
	return linkBase64Encoding.EncodeToString(linkJson), nil
}

func decodeLink(encodedLink string) (Link, error) {
	var link Link
	linkJson, err := linkBase64Encoding.DecodeString(encodedLink)
	if err != nil {
		return link, err
	}
	err = json.Unmarshal(linkJson, &link)
	return link, err
}
