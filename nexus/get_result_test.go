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
	UnimplementedHandler
	timesToBlock int
	resultError  error
	requests     []*GetOperationResultRequest
}

func (h *asyncWithResultHandler) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	return &OperationResponseAsync{
		OperationID: "a/sync",
	}, nil
}

func (h *asyncWithResultHandler) getResult(request *GetOperationResultRequest) (*OperationResponseSync, error) {
	if h.resultError != nil {
		return nil, h.resultError
	}
	return &OperationResponseSync{
		Header: request.HTTPRequest.Header,
		Body:   bytes.NewReader([]byte("body")),
	}, nil
}

func (h *asyncWithResultHandler) GetOperationResult(ctx context.Context, request *GetOperationResultRequest) (*OperationResponseSync, error) {
	h.requests = append(h.requests, request)

	if request.HTTPRequest.Header.Get("User-Agent") != userAgent {
		return nil, newBadRequestError("invalid 'User-Agent' header: %q", request.HTTPRequest.Header.Get("User-Agent"))
	}
	if request.HTTPRequest.Header.Get("Content-Type") != "" {
		return nil, newBadRequestError("'Content-Type' header set on request")
	}
	if request.Wait == 0 {
		return h.getResult(request)
	}
	if request.Wait > 0 {
		deadline, set := ctx.Deadline()
		if !set {
			return nil, newBadRequestError("context deadline unset")
		}
		timeout := time.Until(deadline)
		diff := (getResultMaxTimeout - timeout).Abs()
		if diff > time.Millisecond*100 {
			return nil, newBadRequestError("context deadline invalid, timeout: %v", timeout)
		}
	}
	if len(h.requests) <= h.timesToBlock {
		ctx, cancel := context.WithTimeout(ctx, request.Wait)
		defer cancel()
		<-ctx.Done()
		return nil, ErrOperationStillRunning
	}
	return h.getResult(request)
}

func TestWaitResult(t *testing.T) {
	handler := asyncWithResultHandler{timesToBlock: 1}
	ctx, client, teardown := setup(t, &handler)
	defer teardown()

	response, err := client.ExecuteOperation(ctx, ExecuteOperationOptions{
		Operation: "f/o/o",
		Header: http.Header{
			"foo":          []string{"bar"},
			"Content-Type": []string{"checking that this gets unset in the get-result request"},
		},
	})
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, "bar", response.Header.Get("foo"))
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("body"), body)

	require.Equal(t, 2, len(handler.requests))
	require.InDelta(t, testTimeout+getResultContextPadding, handler.requests[0].Wait, float64(time.Millisecond*50))
	require.InDelta(t, testTimeout+getResultContextPadding-getResultMaxTimeout, handler.requests[1].Wait, float64(time.Millisecond*50))
	require.Equal(t, "f/o/o", handler.requests[0].Operation)
	require.Equal(t, "a/sync", handler.requests[0].OperationID)
}

func TestWaitResult_StillRunning(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{timesToBlock: 1000})
	defer teardown()

	result, err := client.StartOperation(ctx, StartOperationOptions{Operation: "foo"})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	ctx = context.Background()
	_, err = handle.GetResult(ctx, GetOperationResultOptions{Wait: time.Millisecond * 200})
	require.ErrorIs(t, err, ErrOperationStillRunning)
}

func TestWaitResult_DeadlineExceeded(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{timesToBlock: 1000})
	defer teardown()

	result, err := client.StartOperation(ctx, StartOperationOptions{Operation: "foo"})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()
	_, err = handle.GetResult(ctx, GetOperationResultOptions{Wait: time.Second})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestPeekResult_StillRunning(t *testing.T) {
	handler := asyncWithResultHandler{resultError: ErrOperationStillRunning}
	ctx, client, teardown := setup(t, &handler)
	defer teardown()

	handle, err := client.NewHandle("foo", "a/sync")
	require.NoError(t, err)
	response, err := handle.GetResult(ctx, GetOperationResultOptions{})
	require.ErrorIs(t, err, ErrOperationStillRunning)
	require.Nil(t, response)
	require.Equal(t, 1, len(handler.requests))
	require.Equal(t, time.Duration(0), handler.requests[0].Wait)
}

func TestPeekResult_Success(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{})
	defer teardown()

	handle, err := client.NewHandle("foo", "a/sync")
	require.NoError(t, err)
	response, err := handle.GetResult(ctx, GetOperationResultOptions{})
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("body"), body)
}

func TestPeekResult_Canceled(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{resultError: &UnsuccessfulOperationError{State: OperationStateCanceled}})
	defer teardown()

	handle, err := client.NewHandle("foo", "a/sync")
	require.NoError(t, err)
	_, err = handle.GetResult(ctx, GetOperationResultOptions{})
	var unsuccessfulOperationError *UnsuccessfulOperationError
	require.ErrorAs(t, err, &unsuccessfulOperationError)
	require.Equal(t, OperationStateCanceled, unsuccessfulOperationError.State)
}
