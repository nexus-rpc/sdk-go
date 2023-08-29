package nexus

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type asyncWithResultHandler struct {
	expectWait time.Duration
	UnimplementedHandler
	timesCalled int
	block       bool
}

func (h *asyncWithResultHandler) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	return &OperationResponseAsync{
		OperationID: "async",
	}, nil
}

func (h *asyncWithResultHandler) GetOperationResult(ctx context.Context, request *GetOperationResultRequest) (OperationResponse, error) {
	if request.HTTPRequest.Header.Get("User-Agent") != userAgent {
		return nil, newBadRequestError("invalid 'User-Agent' header: %q", request.HTTPRequest.Header.Get("User-Agent"))
	}
	h.timesCalled++
	if h.expectWait == 0 {
		if request.Wait {
			return nil, newBadRequestError("request.Wait set")
		}
	} else {
		if !request.Wait {
			return nil, newBadRequestError("request.Wait unset")
		}
		deadline, set := ctx.Deadline()
		if !set {
			return nil, newBadRequestError("context deadline unset")
		}
		timeout := time.Until(deadline)
		diff := (h.expectWait - timeout).Abs()
		if diff > time.Second*2 {
			return nil, newBadRequestError("context deadline invalid")
		}
	}
	if h.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if h.timesCalled < 2 {
		return &OperationResponseAsync{
			OperationID: "async",
		}, nil
	}
	return &OperationResponseSync{
		Header: request.HTTPRequest.Header,
		Body:   bytes.NewReader([]byte("body")),
	}, nil
}

func TestWaitResult(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: testTimeout})
	defer teardown()

	response, err := client.ExecuteOperation(ctx, ExecuteOperationOptions{
		Operation:       "foo",
		GetResultHeader: http.Header{"foo": []string{"bar"}},
	})
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, "bar", response.Header.Get("foo"))
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("body"), body)
}

func TestPeekResult(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: 0})
	defer teardown()

	result, err := client.StartOperation(ctx, StartOperationOptions{Operation: "foo"})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)
	response, err := handle.GetResult(ctx, GetResultOptions{})
	require.ErrorIs(t, err, ErrOperationStillRunning)
	require.Nil(t, response)
}

func TestPeekResult_HandleFromClient(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: 0})
	defer teardown()

	handle, err := client.NewHandle("foo", "async")
	require.NoError(t, err)
	response, err := handle.GetResult(ctx, GetResultOptions{})
	require.ErrorIs(t, err, ErrOperationStillRunning)
	require.Nil(t, response)
}

func TestWaitResult_NoContextDeadline_WaitCapped(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: time.Minute})
	defer teardown()

	result, err := client.StartOperation(ctx, StartOperationOptions{Operation: "foo"})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	ctx = context.Background()
	_, err = handle.GetResult(ctx, GetResultOptions{Wait: time.Minute * 10})
	require.NoError(t, err)
}

func TestWaitResult_DeadlineCapsWaitTime(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: time.Second * 30})
	defer teardown()

	result, err := client.StartOperation(ctx, StartOperationOptions{Operation: "foo"})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	_, err = handle.GetResult(ctx, GetResultOptions{Wait: time.Hour})
	require.NoError(t, err)
}

func TestWaitResult_Timeout(t *testing.T) {
	waitTimeout := 500 * time.Millisecond
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: waitTimeout, block: true})
	defer teardown()

	handle, err := client.NewHandle("foo", "async")
	require.NoError(t, err)
	_, err = handle.GetResult(ctx, GetResultOptions{
		Wait: waitTimeout,
	})
	require.ErrorIs(t, err, ErrOperationStillRunning)
}
