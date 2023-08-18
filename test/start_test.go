package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexusapi"
	"github.com/nexus-rpc/sdk-go/nexusclient"
	"github.com/nexus-rpc/sdk-go/nexusserver"
	"github.com/stretchr/testify/require"
)

const testTimeout = time.Second * 5

type successHandler struct {
	unimplementedHandler
}

func (h *successHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	if request.Operation != "foo" {
		return nil, fmt.Errorf("unexpected operation: %s", request.Operation)
	}
	if request.CallbackURL != "http://test/callback" {
		return nil, fmt.Errorf("unexpected callback URL: %s", request.CallbackURL)
	}

	return &nexusserver.OperationResponseSync{
		Header:  request.HTTPRequest.Header.Clone(),
		Content: request.HTTPRequest.Body,
	}, nil
}

func TestSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client, listener := setup(t, &successHandler{})
	defer listener.Close()

	requestBody := []byte{0x00, 0x01}

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
		Operation:   "foo",
		CallbackURL: "http://test/callback",
		Header:      http.Header{"Echo": []string{"test"}},
		Body:        bytes.NewReader(requestBody),
	})
	require.NoError(t, err)
	require.Equal(t, "", handle.ID())
	require.Equal(t, nexusapi.OperationStateSucceeded, handle.State())
	response, err := handle.Result(ctx)
	require.NoError(t, err)
	defer handle.Close()
	require.Equal(t, "test", response.Header.Get("Echo"))
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, requestBody, responseBody)
}

type requestIDEchoHandler struct {
	unimplementedHandler
}

func (h *requestIDEchoHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return nexusserver.NewBytesOperationResultSync(nil, []byte(request.RequestID))
}

func TestClientRequestID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client, listener := setup(t, &requestIDEchoHandler{})
	defer listener.Close()

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
			handle, err := client.StartOperation(ctx, c.request)
			require.NoError(t, err)
			response, err := handle.Result(ctx)
			require.NoError(t, err)
			defer handle.Close()
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
	return nexusserver.NewJSONOperationResultSync(nil, "success")
}

func TestJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client, listener := setup(t, &jsonHandler{})
	defer listener.Close()

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
		Operation: "foo",
	})
	require.NoError(t, err)
	defer handle.Close()
	response, err := handle.Result(ctx)
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
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client, listener := setup(t, &asyncHandler{})
	defer listener.Close()

	handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
		Operation: "foo",
	})
	require.NoError(t, err)
	defer handle.Close()
	require.Equal(t, nexusapi.OperationStateRunning, handle.State())
}

type unsuccessfulHandler struct {
	unimplementedHandler
}

func (h *unsuccessfulHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return nil, &nexusapi.UnsuccessfulOperationError{
		// We're passing the desired state via request ID in this test.
		State: nexusapi.OperationState(request.RequestID),
		Failure: nexusapi.Failure{
			Message: "intentional",
		},
	}
}

func TestUnsuccessful(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client, listener := setup(t, &unsuccessfulHandler{})
	defer listener.Close()

	cases := []string{"canceled", "failed"}
	for _, c := range cases {
		handle, err := client.StartOperation(ctx, nexusclient.StartOperationRequest{
			Operation: "foo",
			RequestID: c,
		})
		require.NoError(t, err)
		defer handle.Close()
		_, err = handle.Result(ctx)
		var unsuccessfulError *nexusapi.UnsuccessfulOperationError
		require.ErrorAs(t, err, &unsuccessfulError)
		require.Equal(t, nexusapi.OperationState(c), unsuccessfulError.State)
	}
}

func setup(t *testing.T, handler nexusserver.Handler) (*nexusclient.Client, io.Closer) {
	httpHandler := nexusserver.NewHTTPHandler(nexusserver.Options{
		Handler: handler,
	})

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	client, err := nexusclient.NewClient(nexusclient.Options{
		ServiceBaseURL: fmt.Sprintf("http://%s/", listener.Addr().String()),
	})
	require.NoError(t, err)

	go http.Serve(listener, httpHandler)

	return client, listener
}
