package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexusclient"
	"github.com/nexus-rpc/sdk-go/nexusserver"
	"github.com/stretchr/testify/require"
)

type asyncWithCancelHandler struct {
	expectHeader bool
	unimplementedHandler
}

func (h *asyncWithCancelHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return &nexusserver.OperationResponseAsync{
		OperationID: "async",
	}, nil
}

func (h *asyncWithCancelHandler) CancelOperation(ctx context.Context, request *nexusserver.CancelOperationRequest) error {
	if request.Operation != "foo" {
		return newBadRequestError("expected operation to be 'foo', got: %s", request.Operation)
	}
	if request.OperationID != "async" {
		return newBadRequestError("expected operation ID to be 'async', got: %s", request.OperationID)
	}
	if h.expectHeader && request.HTTPRequest.Header.Get("foo") != "bar" {
		return newBadRequestError("invalid 'foo' request header")
	}
	if request.HTTPRequest.Header.Get("User-Agent") != nexusclient.UserAgent {
		return newBadRequestError("invalid 'User-Agent' header: %q", request.HTTPRequest.Header.Get("User-Agent"))
	}
	return nil
}

func TestCancel_HandleFromStart(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithCancelHandler{expectHeader: true})
	defer teardown()

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
		Operation: "foo",
	})
	require.NoError(t, err)
	defer handle.Close()
	err = handle.Cancel(ctx, nexusclient.CancelOptions{
		Header: http.Header{"foo": []string{"bar"}},
	})
	require.NoError(t, err)
}

func TestCancel_HandleFromClient(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithCancelHandler{})
	defer teardown()

	handle := client.GetHandle("foo", "async")
	defer handle.Close()
	err := handle.Cancel(ctx, nexusclient.CancelOptions{})
	require.NoError(t, err)
}
