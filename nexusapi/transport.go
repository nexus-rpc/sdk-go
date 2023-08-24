package nexusapi

import (
	"encoding/json"
	"mime"
	"net/http"
)

// Marshaler takes any value and returns bytes.
// Used in the SDK to customize the default JSON marshaling behavior.
type Marshaler = func(v any) ([]byte, error)

const (
	// Content-Type header.
	HeaderContentType = "Content-Type"
	// Nexus-Operation-State header.
	HeaderOperationState = "Nexus-Operation-State"
	// Nexus-Request-Id header.
	HeaderRequestID = "Nexus-Request-Id"

	// media type for application/json.
	ContentTypeJSON = "application/json"

	// Query param for passing a callback URL.
	QueryCallbackURL = "callback"

	// Query param for passing wait duration.
	QueryWait = "wait"
)

// DefaultMarshaler marshals a value into two space indented JSON.
func DefaultMarshaler(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// IsContentTypeJSON returns true if header contains a parsable Content-Type header with media type of application/json.
func IsContentTypeJSON(header http.Header) bool {
	contentType := header.Get(HeaderContentType)
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == ContentTypeJSON
}
