package test

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexusapi"
	"github.com/nexus-rpc/sdk-go/nexusclient"
	"github.com/nexus-rpc/sdk-go/nexusserver"
	"github.com/stretchr/testify/require"
)

type asyncWithResultHandler struct {
	expectWait time.Duration
	unimplementedHandler
	timesCalled int
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
	if h.timesCalled < 2 {
		return &nexusserver.OperationResponseAsync{
			OperationID: "async",
		}, nil
	}
	return nexusserver.NewBytesOperationResultSync(request.HTTPRequest.Header, []byte("body"))
}

func TestWaitResult(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: testTimeout})
	defer teardown()

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{Operation: "foo"})
	require.NoError(t, err)
	defer handle.Close()
	response, err := handle.GetResult(ctx, nexusclient.GetResultOptions{
		Wait:   true,
		Header: http.Header{"foo": []string{"bar"}},
	})
	require.NoError(t, err)
	require.Equal(t, nexusapi.OperationStateSucceeded, handle.State())
	require.Equal(t, "bar", response.Header.Get("foo"))
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("body"), body)
}

func TestPeekResult(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: 0})
	defer teardown()

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{Operation: "foo"})
	require.NoError(t, err)
	defer handle.Close()
	response, err := handle.GetResult(ctx, nexusclient.GetResultOptions{})
	require.NoError(t, err)
	require.Equal(t, nexusapi.OperationStateRunning, handle.State())
	require.Nil(t, response)
}

func TestPeekResultHandleFromClient(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: 0})
	defer teardown()

	handle := client.GetHandle("foo", "async")
	require.Equal(t, nexusapi.OperationState(""), handle.State())
	defer handle.Close()
	response, err := handle.GetResult(ctx, nexusclient.GetResultOptions{})
	require.NoError(t, err)
	require.Equal(t, nexusapi.OperationStateRunning, handle.State())
	require.Nil(t, response)
}

func TestWaitResultNoDeadline(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: time.Minute})
	defer teardown()

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{Operation: "foo"})
	require.NoError(t, err)
	defer handle.Close()

	ctx = context.Background()
	_, err = handle.GetResult(ctx, nexusclient.GetResultOptions{Wait: true})
	require.NoError(t, err)
}

func TestWaitResultCappedDeadline(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithResultHandler{expectWait: time.Minute})
	defer teardown()

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{Operation: "foo"})
	require.NoError(t, err)
	defer handle.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()
	_, err = handle.GetResult(ctx, nexusclient.GetResultOptions{Wait: true})
	require.NoError(t, err)
}
