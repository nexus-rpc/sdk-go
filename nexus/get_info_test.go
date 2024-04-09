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

func (h *asyncWithInfoHandler) StartOperation(ctx context.Context, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return &HandlerStartOperationResultAsync{
		OperationID: "needs /URL/ escaping",
	}, nil
}

func (h *asyncWithInfoHandler) GetOperationInfo(ctx context.Context, operation, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	if operation != "escape/me" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "expected operation to be 'escape me', got: %s", operation)
	}
	if operationID != "needs /URL/ escaping" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "expected operation ID to be 'needs URL escaping', got: %s", operationID)
	}
	if h.expectHeader && options.Header.Get("test") != "ok" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'test' request header")
	}
	if options.Header.Get("User-Agent") != userAgent {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'User-Agent' header: %q", options.Header.Get("User-Agent"))
	}
	return &OperationInfo{
		ID:    operationID,
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
	require.Equal(t, handle.ID, info.ID)
	require.Equal(t, OperationStateCanceled, info.State)
}

func TestGetInfoHandleFromClientNoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{})
	defer teardown()

	handle, err := client.NewHandle("escape/me", "needs /URL/ escaping")
	require.NoError(t, err)
	info, err := handle.GetInfo(ctx, GetOperationInfoOptions{})
	require.NoError(t, err)
	require.Equal(t, handle.ID, info.ID)
	require.Equal(t, OperationStateCanceled, info.State)
}

type asyncWithInfoTimeoutHandler struct {
	UnimplementedHandler
}

func (h *asyncWithInfoTimeoutHandler) StartOperation(ctx context.Context, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return &HandlerStartOperationResultAsync{
		OperationID: "needs /URL/ escaping",
	}, nil
}

func (h *asyncWithInfoTimeoutHandler) GetOperationInfo(ctx context.Context, operation, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	time.Sleep(20 * time.Millisecond)

	if ctx.Err() != nil {
		return nil, HandlerErrorf(HandlerErrorTypeDownstreamTimeout, "handler exceeded request timeout of %s", options.Header.Get(headerRequestTimeout))
	}

	return &OperationInfo{
		ID:    operationID,
		State: OperationStateCanceled,
	}, nil
}

func TestGetInfoHandlerTimeout(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoTimeoutHandler{})
	defer teardown()

	type testcase struct {
		name         string
		timeout      time.Duration
		setOnHeader  bool
		setOnContext bool
		validator    func(t *testing.T, handle *OperationHandle[*LazyValue], info *OperationInfo, err error)
	}
	cases := []testcase{
		{
			name:         "time_out: set on context",
			timeout:      1 * time.Millisecond,
			setOnHeader:  false,
			setOnContext: true,
			validator: func(t *testing.T, handle *OperationHandle[*LazyValue], info *OperationInfo, err error) {
				require.ErrorContains(t, err, "context deadline exceeded")
			},
		},
		{
			name:         "time_out: set on header",
			timeout:      1 * time.Millisecond,
			setOnHeader:  true,
			setOnContext: false,
			validator: func(t *testing.T, handle *OperationHandle[*LazyValue], info *OperationInfo, err error) {
				require.ErrorContains(t, err, "handler exceeded request timeout of 1ms")
			},
		},
		{
			name:         "time_out: set on context and header",
			timeout:      1 * time.Millisecond,
			setOnHeader:  true,
			setOnContext: true,
			validator: func(t *testing.T, handle *OperationHandle[*LazyValue], info *OperationInfo, err error) {
				require.ErrorContains(t, err, "context deadline exceeded")
			},
		},
		{
			name:         "success: set on context",
			timeout:      5 * time.Second,
			setOnHeader:  false,
			setOnContext: true,
			validator: func(t *testing.T, handle *OperationHandle[*LazyValue], info *OperationInfo, err error) {
				require.NoError(t, err)
				require.Equal(t, handle.ID, info.ID)
				require.Equal(t, OperationStateCanceled, info.State)
			},
		},
		{
			name:         "success: set on header",
			timeout:      5 * time.Second,
			setOnHeader:  true,
			setOnContext: false,
			validator: func(t *testing.T, handle *OperationHandle[*LazyValue], info *OperationInfo, err error) {
				require.NoError(t, err)
				require.Equal(t, handle.ID, info.ID)
				require.Equal(t, OperationStateCanceled, info.State)
			},
		},
		{
			name:         "success: set on context and header",
			timeout:      5 * time.Second,
			setOnHeader:  true,
			setOnContext: false,
			validator: func(t *testing.T, handle *OperationHandle[*LazyValue], info *OperationInfo, err error) {
				require.NoError(t, err)
				require.Equal(t, handle.ID, info.ID)
				require.Equal(t, OperationStateCanceled, info.State)
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			handle, err := client.NewHandle("escape/me", "needs /URL/ escaping")
			require.NoError(t, err)

			requestCtx := ctx
			if c.setOnContext {
				var cancel context.CancelFunc
				requestCtx, cancel = context.WithTimeout(ctx, c.timeout)
				defer cancel()
			}
			opts := GetOperationInfoOptions{}
			if c.setOnHeader {
				opts.Header = Header{headerRequestTimeout: c.timeout.String()}
			}

			info, err := handle.GetInfo(requestCtx, opts)
			c.validator(t, handle, info, err)
		})
	}
}
