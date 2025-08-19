package nexus

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFailure_GenericError(t *testing.T) {
	h := baseHTTPHandler{
		logger:           slog.Default(),
		failureConverter: defaultFailureConverter,
	}

	writer := httptest.NewRecorder()
	h.writeFailure(writer, fmt.Errorf("foo"))

	require.Equal(t, http.StatusInternalServerError, writer.Code)
	require.Equal(t, contentTypeJSON, writer.Header().Get("Content-Type"))

	var failure *Failure
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &failure))
	require.Equal(t, "internal server error", failure.Message)
}

func TestWriteFailure_HandlerError(t *testing.T) {
	h := baseHTTPHandler{
		logger:           slog.Default(),
		failureConverter: defaultFailureConverter,
	}

	writer := httptest.NewRecorder()
	he := HandlerErrorf(HandlerErrorTypeBadRequest, "foo")
	h.writeFailure(writer, he)

	require.Equal(t, http.StatusBadRequest, writer.Code)
	require.Equal(t, contentTypeJSON, writer.Header().Get("Content-Type"))

	var failure Failure
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &failure))
	actual, err := defaultFailureConverter.FailureToError(failure)
	require.NoError(t, err)
	// Assign the original failure object before performing the comparison.
	he.OriginalFailure = &failure
	require.Equal(t, he, actual)
}

func TestWriteFailure_OperationError(t *testing.T) {
	h := baseHTTPHandler{
		logger:           slog.Default(),
		failureConverter: defaultFailureConverter,
	}

	writer := httptest.NewRecorder()
	oe := NewOperationCanceledError("canceled")
	h.writeFailure(writer, oe)

	require.Equal(t, statusOperationFailed, writer.Code)
	require.Equal(t, contentTypeJSON, writer.Header().Get("Content-Type"))
	require.Equal(t, string(OperationStateCanceled), writer.Header().Get(headerOperationState))

	var failure Failure
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &failure))
	actual, err := defaultFailureConverter.FailureToError(failure)
	require.NoError(t, err)
	// Assign the original failure object before performing the comparison.
	oe.OriginalFailure = &failure
	require.Equal(t, oe, actual)
}
