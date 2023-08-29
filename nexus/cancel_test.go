package nexus

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type asyncWithCancelHandler struct {
	expectHeader bool
	UnimplementedHandler
}

func (h *asyncWithCancelHandler) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	return &OperationResponseAsync{
		OperationID: "async",
	}, nil
}

func (h *asyncWithCancelHandler) CancelOperation(ctx context.Context, request *CancelOperationRequest) error {
	if request.Operation != "foo" {
		return newBadRequestError("expected operation to be 'foo', got: %s", request.Operation)
	}
	if request.OperationID != "async" {
		return newBadRequestError("expected operation ID to be 'async', got: %s", request.OperationID)
	}
	if h.expectHeader && request.HTTPRequest.Header.Get("foo") != "bar" {
		return newBadRequestError("invalid 'foo' request header")
	}
	if request.HTTPRequest.Header.Get("User-Agent") != userAgent {
		return newBadRequestError("invalid 'User-Agent' header: %q", request.HTTPRequest.Header.Get("User-Agent"))
	}
	return nil
}

func TestCancel_HandleFromStart(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithCancelHandler{expectHeader: true})
	defer teardown()

	result, err := client.StartOperation(ctx, StartOperationOptions{
		Operation: "foo",
	})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)
	err = handle.Cancel(ctx, CancelOptions{
		Header: http.Header{"foo": []string{"bar"}},
	})
	require.NoError(t, err)
}

func TestCancel_HandleFromClient(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithCancelHandler{})
	defer teardown()

	handle, err := client.NewHandle("foo", "async")
	require.NoError(t, err)
	err = handle.Cancel(ctx, CancelOptions{})
	require.NoError(t, err)
}
