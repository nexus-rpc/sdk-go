package nexus

import (
	"fmt"
	"net/http"
)

func newBadRequestError(message string, args ...any) error {
	return &HandlerError{StatusCode: http.StatusBadRequest, Failure: &Failure{Message: fmt.Sprintf(message, args...)}}
}
