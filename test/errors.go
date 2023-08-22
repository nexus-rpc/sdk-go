package test

import (
	"fmt"
	"net/http"

	"github.com/nexus-rpc/sdk-go/nexusapi"
	"github.com/nexus-rpc/sdk-go/nexusserver"
)

func newBadRequestError(message string, args ...any) error {
	return &nexusserver.HandlerError{StatusCode: http.StatusBadRequest, Failure: &nexusapi.Failure{Message: fmt.Sprintf(message, args...)}}
}
