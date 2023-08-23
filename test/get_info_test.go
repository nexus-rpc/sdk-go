package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexusapi"
	"github.com/nexus-rpc/sdk-go/nexusclient"
	"github.com/nexus-rpc/sdk-go/nexusserver"
	"github.com/stretchr/testify/require"
)

type asyncWithInfoHandler struct {
	unimplementedHandler
	expectHeader bool
}

func (h *asyncWithInfoHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return &nexusserver.OperationResponseAsync{
		OperationID: "async",
	}, nil
}

func (h *asyncWithInfoHandler) GetOperationInfo(ctx context.Context, request *nexusserver.GetOperationInfoRequest) (*nexusapi.OperationInfo, error) {
	if request.Operation != "foo" {
		return nil, newBadRequestError("expected operation to be 'foo', got: %s", request.Operation)
	}
	if request.OperationID != "async" {
		return nil, newBadRequestError("expected operation ID to be 'async', got: %s", request.OperationID)
	}
	if h.expectHeader && request.HTTPRequest.Header.Get("foo") != "bar" {
		return nil, newBadRequestError("invalid 'foo' request header")
	}
	if request.HTTPRequest.Header.Get("User-Agent") != nexusclient.UserAgent {
		return nil, newBadRequestError("invalid 'User-Agent' header: %q", request.HTTPRequest.Header.Get("User-Agent"))
	}
	return &nexusapi.OperationInfo{
		ID:    request.OperationID,
		State: nexusapi.OperationStateCanceled,
	}, nil
}

func TestGetHandlerFromStartInfoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{expectHeader: true})
	defer teardown()

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
		Operation: "foo",
	})
	require.NoError(t, err)
	defer handle.Close()
	info, err := handle.GetInfo(ctx, nexusclient.GetInfoOptions{
		Header: http.Header{"foo": []string{"bar"}},
	})
	require.NoError(t, err)
	require.Equal(t, handle.ID(), info.ID)
	require.Equal(t, nexusapi.OperationStateCanceled, info.State)
}

func TestGetInfoHandleFromClientNoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{})
	defer teardown()

	handle := client.GetHandle("foo", "async")
	info, err := handle.GetInfo(ctx, nexusclient.GetInfoOptions{})
	require.NoError(t, err)
	require.Equal(t, handle.ID(), info.ID)
	require.Equal(t, nexusapi.OperationStateCanceled, info.State)
}
