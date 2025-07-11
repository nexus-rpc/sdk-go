package nexus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type asyncWithInfoHandler struct {
	UnimplementedHandler
	expectHeader bool
}

func (h *asyncWithInfoHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return &HandlerStartOperationResultAsync{
		OperationToken: "just-a-token",
	}, nil
}

func (h *asyncWithInfoHandler) GetOperationInfo(ctx context.Context, service, operation, token string, options GetOperationInfoOptions) (*OperationInfo, error) {
	if service != testService {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "unexpected service: %s", service)
	}
	if operation != "escape/me" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "expected operation to be 'escape me', got: %s", operation)
	}
	if token != "just-a-token" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "expected operation token to be 'just-a-token', got: %s", token)
	}
	if h.expectHeader && options.Header.Get("test") != "ok" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'test' request header")
	}
	if options.Header.Get("User-Agent") != userAgent {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'User-Agent' header: %q", options.Header.Get("User-Agent"))
	}
	return &OperationInfo{
		ID:    token,
		State: OperationStateCanceled,
	}, nil
}

func TestGetHandlerFromStartInfoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{expectHeader: true})
	defer teardown()

	result, err := client.StartOperation(ctx, "escape/me", nil, StartOperationOptions{})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)
	info, err := handle.GetInfo(ctx, GetOperationInfoOptions{
		Header: Header{"test": "ok"},
	})
	require.NoError(t, err)
	require.Equal(t, handle.Token, info.Token)
	require.Equal(t, OperationStateCanceled, info.State)
}

func TestGetInfoHandleFromClientNoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{})
	defer teardown()

	handle, err := client.NewHandle("escape/me", "just-a-token")
	require.NoError(t, err)
	info, err := handle.GetInfo(ctx, GetOperationInfoOptions{})
	require.NoError(t, err)
	require.Equal(t, handle.ID, info.ID)
	require.Equal(t, OperationStateCanceled, info.State)
}

type asyncWithInfoTimeoutHandler struct {
	expectedTimeout time.Duration
	UnimplementedHandler
}

func (h *asyncWithInfoTimeoutHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return &HandlerStartOperationResultAsync{
		OperationToken: "timeout",
	}, nil
}

func (h *asyncWithInfoTimeoutHandler) GetOperationInfo(ctx context.Context, service, operation, token string, options GetOperationInfoOptions) (*OperationInfo, error) {
	deadline, set := ctx.Deadline()
	if h.expectedTimeout > 0 && !set {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "expected operation to have timeout set but context has no deadline")
	}
	if h.expectedTimeout <= 0 && set {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "expected operation to have no timeout but context has deadline set")
	}
	timeout := time.Until(deadline)
	if timeout > h.expectedTimeout {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "operation has timeout (%s) greater than expected (%s)", timeout.String(), h.expectedTimeout.String())
	}

	return &OperationInfo{
		ID:    token,
		State: OperationStateCanceled,
	}, nil
}

func TestGetInfo_ContextDeadlinePropagated(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoTimeoutHandler{expectedTimeout: testTimeout})
	defer teardown()

	handle, err := client.NewHandle("foo", "timeout")
	require.NoError(t, err)
	_, err = handle.GetInfo(ctx, GetOperationInfoOptions{})
	require.NoError(t, err)
}

func TestGetInfo_RequestTimeoutHeaderOverridesContextDeadline(t *testing.T) {
	timeout := 100 * time.Millisecond
	// relies on ctx returned here having default testTimeout set greater than expected timeout
	ctx, client, teardown := setup(t, &asyncWithInfoTimeoutHandler{expectedTimeout: timeout})
	defer teardown()

	handle, err := client.NewHandle("foo", "timeout")
	require.NoError(t, err)
	_, err = handle.GetInfo(ctx, GetOperationInfoOptions{Header: Header{HeaderRequestTimeout: formatDuration(timeout)}})
	require.NoError(t, err)
}

func TestGetInfo_TimeoutNotPropagated(t *testing.T) {
	_, client, teardown := setup(t, &asyncWithInfoTimeoutHandler{})
	defer teardown()

	handle, err := client.NewHandle("foo", "timeout")
	require.NoError(t, err)
	_, err = handle.GetInfo(context.Background(), GetOperationInfoOptions{})
	require.NoError(t, err)
}
