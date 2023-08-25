package nexus

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexusclient"
	"github.com/nexus-rpc/sdk-go/nexusserver"
	"github.com/stretchr/testify/require"
)

type asyncWithResultHandler struct {
	expectWait time.Duration
	unimplementedHandler
	timesCalled int
	block       bool
}

func (h *asyncWithResultHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return &nexusserver.OperationResponseAsync{
		OperationID: "async",
	}, nil
}

func (h *asyncWithResultHandler) GetOperationResult(ctx context.Context, request *nexusserver.GetOperationResultRequest) (nexusserver.OperationResponse, error) {
	if request.HTTPRequest.Header.Get("User-Agent") != nexusclient.UserAgent {
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
		return &nexusserver.OperationResponseAsync{
			OperationID: "async",
		}, nil
	}
	return &nexusserver.OperationResponseSync{
		Header: request.HTTPRequest.Header,
		Body:   bytes.NewReader([]byte("body")),
	}, nil
}

func TestWaitResult(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: testTimeout})
	defer teardown()

	response, err := client.ExecuteOperation(ctx, nexusclient.ExecuteOperationRequest{
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

	result, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{Operation: "foo"})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)
	response, err := handle.GetResult(ctx, nexusclient.GetResultOptions{})
	require.NoError(t, err)
	require.Nil(t, response)
}

func TestPeekResult_HandleFromClient(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: 0})
	defer teardown()

	handle := client.NewHandle("foo", "async")
	response, err := handle.GetResult(ctx, nexusclient.GetResultOptions{})
	require.NoError(t, err)
	require.Nil(t, response)
}

func TestWaitResult_NoDeadline(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: time.Minute})
	defer teardown()

	result, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{Operation: "foo"})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	ctx = context.Background()
	_, err = handle.GetResult(ctx, nexusclient.GetResultOptions{Wait: true})
	require.NoError(t, err)
}

func TestWaitResult_CappedDeadline(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: time.Minute})
	defer teardown()

	result, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{Operation: "foo"})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	_, err = handle.GetResult(ctx, nexusclient.GetResultOptions{Wait: true})
	require.NoError(t, err)
}

func TestWaitResult_Timeout(t *testing.T) {
	waitTimeout := 500 * time.Millisecond
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: waitTimeout, block: true})
	defer teardown()

	ctx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()

	handle := client.NewHandle("foo", "async")
	_, err := handle.GetResult(ctx, nexusclient.GetResultOptions{
		Wait: true,
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
