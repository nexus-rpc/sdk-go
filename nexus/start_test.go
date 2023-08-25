package nexus

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexusapi"
	"github.com/nexus-rpc/sdk-go/nexusclient"
	"github.com/nexus-rpc/sdk-go/nexusserver"
	"github.com/stretchr/testify/require"
)

type successHandler struct {
	unimplementedHandler
}

func (h *successHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	if request.Operation != "foo" {
		return nil, newBadRequestError("unexpected operation: %s", request.Operation)
	}
	if request.CallbackURL != "http://test/callback" {
		return nil, newBadRequestError("unexpected callback URL: %s", request.CallbackURL)
	}
	if request.HTTPRequest.Header.Get("User-Agent") != nexusclient.UserAgent {
		return nil, newBadRequestError("invalid 'User-Agent' header: %q", request.HTTPRequest.Header.Get("User-Agent"))
	}

	return &nexusserver.OperationResponseSync{
		Header: request.HTTPRequest.Header.Clone(),
		Body:   request.HTTPRequest.Body,
	}, nil
}

func TestSuccess(t *testing.T) {
	ctx, client, teardown := setup(t, &successHandler{})
	defer teardown()

	requestBody := []byte{0x00, 0x01}

	response, err := client.ExecuteOperation(ctx, nexusclient.ExecuteOperationRequest{
		Operation:   "foo",
		CallbackURL: "http://test/callback",
		StartHeader: http.Header{"Echo": []string{"test"}},
		Body:        bytes.NewReader(requestBody),
	})
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, "test", response.Header.Get("Echo"))
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, requestBody, responseBody)
}

type requestIDEchoHandler struct {
	unimplementedHandler
}

func (h *requestIDEchoHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return &nexusserver.OperationResponseSync{Body: bytes.NewReader([]byte(request.RequestID))}, nil
}

func TestClientRequestID(t *testing.T) {
	ctx, client, teardown := setup(t, &requestIDEchoHandler{})
	defer teardown()

	type testcase struct {
		name      string
		request   nexusclient.StartOperationRequest
		validator func(*testing.T, []byte)
	}

	cases := []testcase{
		{
			name: "unspecified",
			request: nexusclient.StartOperationRequest{
				Operation: "foo",
			},
			validator: func(t *testing.T, body []byte) {
				_, err := uuid.ParseBytes(body)
				require.NoError(t, err)
			},
		},
		{
			name: "provided directly",
			request: nexusclient.StartOperationRequest{
				Operation: "foo",
				RequestID: "direct",
			},
			validator: func(t *testing.T, body []byte) {
				require.Equal(t, []byte("direct"), body)
			},
		},
		{
			name: "provided via headers",
			request: nexusclient.StartOperationRequest{
				Operation: "foo",
				Header:    http.Header{nexusapi.HeaderRequestID: []string{"via header"}},
			},
			validator: func(t *testing.T, body []byte) {
				require.Equal(t, []byte("via header"), body)
			},
		},
		{
			name: "direct overwrites headers",
			request: nexusclient.StartOperationRequest{
				Operation: "foo",
				RequestID: "direct",
				Header:    http.Header{nexusapi.HeaderRequestID: []string{"via header"}},
			},
			validator: func(t *testing.T, body []byte) {
				require.Equal(t, []byte("direct"), body)
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			result, err := client.StartOperation(ctx, c.request)
			require.NoError(t, err)
			response := result.Successful
			require.NotNil(t, response)
			defer response.Body.Close()
			responseBody, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			c.validator(t, responseBody)
		})
	}
}

type jsonHandler struct {
	unimplementedHandler
}

func (h *jsonHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return nexusserver.NewJSONOperationResponseSync("success")
}

func TestJSON(t *testing.T) {
	ctx, client, teardown := setup(t, &jsonHandler{})
	defer teardown()

	result, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
		Operation: "foo",
	})
	require.NoError(t, err)
	response := result.Successful
	require.Equal(t, "application/json", response.Header.Get("Content-Type"))
	require.NoError(t, err)
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, []byte(`"success"`), responseBody)
}

type asyncHandler struct {
	unimplementedHandler
}

func (h *asyncHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return &nexusserver.OperationResponseAsync{
		OperationID: "async",
	}, nil
}

func TestAsync(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncHandler{})
	defer teardown()

	result, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
		Operation: "foo",
	})
	require.NoError(t, err)
	require.NotNil(t, result.Pending)
}

type unsuccessfulHandler struct {
	unimplementedHandler
}

func (h *unsuccessfulHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return nil, &nexusapi.UnsuccessfulOperationError{
		// We're passing the desired state via request ID in this test.
		State: nexusapi.OperationState(request.RequestID),
		Failure: &nexusapi.Failure{
			Message: "intentional",
		},
	}
}

func TestUnsuccessful(t *testing.T) {
	ctx, client, teardown := setup(t, &unsuccessfulHandler{})
	defer teardown()

	cases := []string{"canceled", "failed"}
	for _, c := range cases {
		_, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
			Operation: "foo",
			RequestID: c,
		})
		var unsuccessfulError *nexusapi.UnsuccessfulOperationError
		require.ErrorAs(t, err, &unsuccessfulError)
		require.Equal(t, nexusapi.OperationState(c), unsuccessfulError.State)
	}
}
