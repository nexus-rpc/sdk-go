package nexusserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexusapi"
	"github.com/stretchr/testify/require"
)

func TestWriteFailure_GenericError(t *testing.T) {
	h := baseHTTPHandler{
		marshaler: nexusapi.DefaultMarshaler,
		logger:    slog.New(newDefaultLogHandler()),
	}

	writer := httptest.NewRecorder()
	h.writeFailure(writer, fmt.Errorf("foo"))

	require.Equal(t, http.StatusInternalServerError, writer.Code)
	require.Equal(t, contentTypeJSON, writer.Header().Get(headerContentType))

	var failure *nexusapi.Failure
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &failure))
	require.Equal(t, "internal server error", failure.Message)
}

func TestWriteFailure_HandlerError(t *testing.T) {
	h := baseHTTPHandler{
		marshaler: nexusapi.DefaultMarshaler,
		logger:    slog.New(newDefaultLogHandler()),
	}

	writer := httptest.NewRecorder()
	h.writeFailure(writer, newBadRequestError("foo"))

	require.Equal(t, http.StatusBadRequest, writer.Code)
	require.Equal(t, contentTypeJSON, writer.Header().Get(headerContentType))

	var failure *nexusapi.Failure
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &failure))
	require.Equal(t, "foo", failure.Message)
}

func TestWriteFailure_UnsuccessfulOperationError(t *testing.T) {
	h := baseHTTPHandler{
		marshaler: nexusapi.DefaultMarshaler,
		logger:    slog.New(newDefaultLogHandler()),
	}

	writer := httptest.NewRecorder()
	h.writeFailure(writer, &nexusapi.UnsuccessfulOperationError{
		State:   nexusapi.OperationStateCanceled,
		Failure: &nexusapi.Failure{Message: "canceled"},
	})

	require.Equal(t, nexusapi.StatusOperationFailed, writer.Code)
	require.Equal(t, contentTypeJSON, writer.Header().Get(headerContentType))
	require.Equal(t, string(nexusapi.OperationStateCanceled), writer.Header().Get(headerOperationState))

	var failure *nexusapi.Failure
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &failure))
	require.Equal(t, "canceled", failure.Message)
}
