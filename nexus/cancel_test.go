package nexus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type asyncWithCancelHandler struct {
	expectHeader bool
	UnimplementedHandler
}

func (h *asyncWithCancelHandler) StartOperation(ctx context.Context, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return &HandlerStartOperationResultAsync{
		OperationID: "a/sync",
	}, nil
}

func (h *asyncWithCancelHandler) CancelOperation(ctx context.Context, operation, operationID string, options CancelOperationOptions) error {
	if operation != "f/o/o" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "expected operation to be 'foo', got: %s", operation)
	}
	if operationID != "a/sync" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "expected operation ID to be 'async', got: %s", operationID)
	}
	if h.expectHeader && options.Header.Get("foo") != "bar" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'foo' request header")
	}
	if options.Header.Get("User-Agent") != userAgent {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'User-Agent' header: %q", options.Header.Get("User-Agent"))
	}
	return nil
}

func TestCancel_HandleFromStart(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithCancelHandler{expectHeader: true})
	defer teardown()

	result, err := client.StartOperation(ctx, "f/o/o", nil, StartOperationOptions{})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)
	err = handle.Cancel(ctx, CancelOperationOptions{
		Header: Header{"foo": "bar"},
	})
	require.NoError(t, err)
}

func TestCancel_HandleFromClient(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithCancelHandler{})
	defer teardown()

	handle, err := client.NewHandle("f/o/o", "a/sync")
	require.NoError(t, err)
	err = handle.Cancel(ctx, CancelOperationOptions{})
	require.NoError(t, err)
}
