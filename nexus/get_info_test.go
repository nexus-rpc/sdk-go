package nexus

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type asyncWithInfoHandler struct {
	UnimplementedHandler
	expectHeader bool
}

func (h *asyncWithInfoHandler) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	return &OperationResponseAsync{
		OperationID: "async",
	}, nil
}

func (h *asyncWithInfoHandler) GetOperationInfo(ctx context.Context, request *GetOperationInfoRequest) (*OperationInfo, error) {
	if request.Operation != "foo" {
		return nil, newBadRequestError("expected operation to be 'foo', got: %s", request.Operation)
	}
	if request.OperationID != "async" {
		return nil, newBadRequestError("expected operation ID to be 'async', got: %s", request.OperationID)
	}
	if h.expectHeader && request.HTTPRequest.Header.Get("foo") != "bar" {
		return nil, newBadRequestError("invalid 'foo' request header")
	}
	if request.HTTPRequest.Header.Get("User-Agent") != userAgent {
		return nil, newBadRequestError("invalid 'User-Agent' header: %q", request.HTTPRequest.Header.Get("User-Agent"))
	}
	return &OperationInfo{
		ID:    request.OperationID,
		State: OperationStateCanceled,
	}, nil
}

func TestGetHandlerFromStartInfoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{expectHeader: true})
	defer teardown()

	result, err := client.StartOperation(ctx, StartOperationOptions{
		Operation: "foo",
	})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)
	info, err := handle.GetInfo(ctx, GetInfoOptions{
		Header: http.Header{"foo": []string{"bar"}},
	})
	require.NoError(t, err)
	require.Equal(t, handle.ID, info.ID)
	require.Equal(t, OperationStateCanceled, info.State)
}

func TestGetInfoHandleFromClientNoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{})
	defer teardown()

	handle, err := client.NewHandle("foo", "async")
	require.NoError(t, err)
	info, err := handle.GetInfo(ctx, GetInfoOptions{})
	require.NoError(t, err)
	require.Equal(t, handle.ID, info.ID)
	require.Equal(t, OperationStateCanceled, info.State)
}
