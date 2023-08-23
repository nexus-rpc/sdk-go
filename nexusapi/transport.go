package nexusapi

import (
	"encoding/json"
	"mime"
	"net/http"
)

type Marshaler = func(v any) ([]byte, error)

const (
	HeaderContentType    = "Content-Type"
	HeaderOperationState = "Nexus-Operation-State"
	HeaderRequestID      = "Nexus-Request-Id"

	ContentTypeJSON = "application/json"

	QueryCallbackURL = "callback"
	QueryWait        = "wait"
)

func DefaultMarshaler(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func IsContentTypeJSON(header http.Header) bool {
	contentType := header.Get(HeaderContentType)
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == ContentTypeJSON
}
