package nexusapi

import (
	"mime"
	"net/http"
)

const (
	HeaderContentType    = "Content-Type"
	HeaderOperationState = "Nexus-Operation-State"
	HeaderRequestID      = "Nexus-Request-Id"

	ContentTypeJSON = "application/json"

	QueryCallbackURL = "callback"
	QueryWait        = "wait"
)

func IsContentTypeJSON(header http.Header) bool {
	contentType := header.Get(HeaderContentType)
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == ContentTypeJSON
}
