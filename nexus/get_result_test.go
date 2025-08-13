package nexus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type request struct {
	options   FetchOperationResultOptions
	operation string
	token     string
	deadline  time.Time
}

type asyncWithResultHandler struct {
	UnimplementedHandler
	timesToBlock     int
	resultError      error
	expectTestHeader bool
	requests         []request
}

func (h *asyncWithResultHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	if h.expectTestHeader && options.Header.Get("test") != "ok" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'test' header: %q", options.Header.Get("test"))
	}

	return &HandlerStartOperationResultAsync{
		OperationToken: "async",
	}, nil
}

func (h *asyncWithResultHandler) fetchResult() (any, error) {
	if h.resultError != nil {
		return nil, h.resultError
	}
	return []byte("body"), nil
}

func (h *asyncWithResultHandler) FetchOperationResult(ctx context.Context, service, operation, token string, options FetchOperationResultOptions) (any, error) {
	req := request{options: options, operation: operation, token: token}
	deadline, set := ctx.Deadline()
	if set {
		req.deadline = deadline
	}
	h.requests = append(h.requests, req)

	if service != testService {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "unexpected service: %s", service)
	}
	if h.expectTestHeader && options.Header.Get("test") != "ok" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'test' header: %q", options.Header.Get("test"))
	}
	if options.Header.Get("User-Agent") != userAgent {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'User-Agent' header: %q", options.Header.Get("User-Agent"))
	}
	if options.Header.Get("Content-Type") != "" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "'Content-Type' header set on request")
	}
	if options.Wait == 0 {
		return h.fetchResult()
	}
	if options.Wait > 0 {
		deadline, set := ctx.Deadline()
		if !set {
			return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "context deadline unset")
		}
		timeout := time.Until(deadline)
		diff := (fetchResultMaxTimeout - timeout).Abs()
		if diff > time.Millisecond*200 {
			return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "context deadline invalid, timeout: %v", timeout)
		}
	}
	if len(h.requests) <= h.timesToBlock {
		ctx, cancel := context.WithTimeout(ctx, options.Wait)
		defer cancel()
		<-ctx.Done()
		return nil, ErrOperationStillRunning
	}
	return h.fetchResult()
}

func TestWaitResult(t *testing.T) {
	handler := asyncWithResultHandler{timesToBlock: 1, expectTestHeader: true}
	ctx, client, teardown := setup(t, &handler)
	defer teardown()

	response, err := client.ExecuteOperation(ctx, "f/o/o", nil, ExecuteOperationOptions{
		Header: Header{"test": "ok"},
	})
	require.NoError(t, err)
	var body []byte
	err = response.Consume(&body)
	require.NoError(t, err)
	require.Equal(t, []byte("body"), body)

	require.Equal(t, 2, len(handler.requests))
	require.InDelta(t, testTimeout+fetchResultContextPadding, handler.requests[0].options.Wait, float64(time.Millisecond*50))
	require.InDelta(t, testTimeout+fetchResultContextPadding-fetchResultMaxTimeout, handler.requests[1].options.Wait, float64(time.Millisecond*50))
	require.Equal(t, "f/o/o", handler.requests[0].operation)
	require.Equal(t, "async", handler.requests[0].token)
}

func TestWaitResult_StillRunning(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{timesToBlock: 1000})
	defer teardown()

	result, err := client.StartOperation(ctx, "foo", nil, StartOperationOptions{})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	ctx = context.Background()
	_, err = handle.FetchResult(ctx, FetchOperationResultOptions{Wait: time.Millisecond * 200})
	require.ErrorIs(t, err, ErrOperationStillRunning)
}

func TestWaitResult_DeadlineExceeded(t *testing.T) {
	handler := &asyncWithResultHandler{timesToBlock: 1000}
	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := client.StartOperation(ctx, "foo", nil, StartOperationOptions{})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()
	deadline, _ := ctx.Deadline()
	_, err = handle.FetchResult(ctx, FetchOperationResultOptions{Wait: time.Second})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	// Allow up to 10 ms delay to account for slow CI.
	// This test is inherently flaky, and should be rewritten.
	require.WithinDuration(t, deadline, handler.requests[0].deadline, 10*time.Millisecond)
}

func TestWaitResult_RequestTimeout(t *testing.T) {
	handler := &asyncWithResultHandler{timesToBlock: 1000}
	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := client.StartOperation(ctx, "foo", nil, StartOperationOptions{})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	timeout := 200 * time.Millisecond
	deadline := time.Now().Add(timeout)
	_, err = handle.FetchResult(ctx, FetchOperationResultOptions{Wait: time.Second, Header: Header{HeaderRequestTimeout: formatDuration(timeout)}})
	require.ErrorIs(t, err, ErrOperationStillRunning)
	require.WithinDuration(t, deadline, handler.requests[0].deadline, 1*time.Millisecond)
}

func TestPeekResult_StillRunning(t *testing.T) {
	handler := asyncWithResultHandler{resultError: ErrOperationStillRunning}
	ctx, client, teardown := setup(t, &handler)
	defer teardown()

	handle, err := client.NewOperationHandle("foo", "a/sync")
	require.NoError(t, err)
	response, err := handle.FetchResult(ctx, FetchOperationResultOptions{})
	require.ErrorIs(t, err, ErrOperationStillRunning)
	require.Nil(t, response)
	require.Equal(t, 1, len(handler.requests))
	require.Equal(t, time.Duration(0), handler.requests[0].options.Wait)
}

func TestPeekResult_Success(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{})
	defer teardown()

	handle, err := client.NewOperationHandle("foo", "a/sync")
	require.NoError(t, err)
	response, err := handle.FetchResult(ctx, FetchOperationResultOptions{})
	require.NoError(t, err)
	var body []byte
	err = response.Consume(&body)
	require.NoError(t, err)
	require.Equal(t, []byte("body"), body)
}

func TestPeekResult_Canceled(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{resultError: &OperationError{State: OperationStateCanceled}})
	defer teardown()

	handle, err := client.NewOperationHandle("foo", "a/sync")
	require.NoError(t, err)
	_, err = handle.FetchResult(ctx, FetchOperationResultOptions{})
	var OperationError *OperationError
	require.ErrorAs(t, err, &OperationError)
	require.Equal(t, OperationStateCanceled, OperationError.State)
}
