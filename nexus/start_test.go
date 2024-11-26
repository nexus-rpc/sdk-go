package nexus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type successHandler struct {
	UnimplementedHandler
}

func (h *successHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	var body []byte
	if err := input.Consume(&body); err != nil {
		return nil, err
	}
	if service != testService {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "unexpected service: %s", service)
	}
	if operation != "i need to/be escaped" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "unexpected operation: %s", operation)
	}
	if options.CallbackURL != "http://test/callback" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "unexpected callback URL: %s", options.CallbackURL)
	}
	if options.CallbackHeader.Get("callback-test") != "ok" {
		return nil, HandlerErrorf(
			HandlerErrorTypeBadRequest,
			"invalid 'callback-test' callback header: %q",
			options.CallbackHeader.Get("callback-test"),
		)
	}
	if options.Header.Get("test") != "ok" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'test' header: %q", options.Header.Get("test"))
	}
	if options.Header.Get("nexus-callback-callback-test") != "" {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "callback header not omitted from options Header")
	}
	if options.Header.Get("User-Agent") != userAgent {
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'User-Agent' header: %q", options.Header.Get("User-Agent"))
	}

	return &HandlerStartOperationResultSync[any]{body}, nil
}

func TestSuccess(t *testing.T) {
	ctx, client, teardown := setup(t, &successHandler{})
	defer teardown()

	requestBody := []byte{0x00, 0x01}

	response, err := client.ExecuteOperation(ctx, "i need to/be escaped", requestBody, ExecuteOperationOptions{
		CallbackURL:    "http://test/callback",
		CallbackHeader: Header{"callback-test": "ok"},
		Header:         Header{"test": "ok"},
	})
	require.NoError(t, err)
	var responseBody []byte
	err = response.Consume(&responseBody)
	require.NoError(t, err)
	require.Equal(t, requestBody, responseBody)
}

type requestIDEchoHandler struct {
	UnimplementedHandler
}

func (h *requestIDEchoHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return &HandlerStartOperationResultSync[any]{
		Value: []byte(options.RequestID),
	}, nil
}

func TestClientRequestID(t *testing.T) {
	ctx, client, teardown := setup(t, &requestIDEchoHandler{})
	defer teardown()

	type testcase struct {
		name      string
		request   StartOperationOptions
		validator func(*testing.T, []byte)
	}

	cases := []testcase{
		{
			name:    "unspecified",
			request: StartOperationOptions{},
			validator: func(t *testing.T, body []byte) {
				_, err := uuid.ParseBytes(body)
				require.NoError(t, err)
			},
		},
		{
			name:    "provided directly",
			request: StartOperationOptions{RequestID: "direct"},
			validator: func(t *testing.T, body []byte) {
				require.Equal(t, []byte("direct"), body)
			},
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			result, err := client.StartOperation(ctx, "foo", nil, c.request)
			require.NoError(t, err)
			response := result.Successful
			require.NotNil(t, response)
			var responseBody []byte
			err = response.Consume(&responseBody)
			require.NoError(t, err)
			c.validator(t, responseBody)
		})
	}
}

type jsonHandler struct {
	UnimplementedHandler
}

func (h *jsonHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	var s string
	if err := input.Consume(&s); err != nil {
		return nil, err
	}
	return &HandlerStartOperationResultSync[any]{Value: s}, nil
}

func TestJSON(t *testing.T) {
	ctx, client, teardown := setup(t, &jsonHandler{})
	defer teardown()

	result, err := client.StartOperation(ctx, "foo", "success", StartOperationOptions{})
	require.NoError(t, err)
	response := result.Successful
	require.NotNil(t, response)
	var operationResult string
	err = response.Consume(&operationResult)
	require.NoError(t, err)
	require.Equal(t, "success", operationResult)
}

type echoHandler struct {
	UnimplementedHandler
}

func (h *echoHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	var output any
	switch options.Header.Get("input-type") {
	case "reader":
		output = input.Reader
	case "content":
		data, err := io.ReadAll(input.Reader)
		if err != nil {
			return nil, err
		}
		output = &Content{
			Header: input.Reader.Header,
			Data:   data,
		}
	}
	return &HandlerStartOperationResultSync[any]{Value: output}, nil
}

func TestReaderIO(t *testing.T) {
	ctx, client, teardown := setup(t, &echoHandler{})
	defer teardown()

	content, err := jsonSerializer{}.Serialize("success")
	require.NoError(t, err)
	reader := &Reader{
		io.NopCloser(bytes.NewReader(content.Data)),
		content.Header,
	}
	testCases := []struct {
		name   string
		input  any
		header Header
	}{
		{
			name:   "content",
			input:  content,
			header: Header{"input-type": "content"},
		},
		{
			name:   "reader",
			input:  reader,
			header: Header{"input-type": "reader"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.StartOperation(ctx, "foo", tc.input, StartOperationOptions{Header: tc.header})
			require.NoError(t, err)
			response := result.Successful
			require.NotNil(t, response)
			var operationResult string
			err = response.Consume(&operationResult)
			require.NoError(t, err)
			require.Equal(t, "success", operationResult)
		})
	}
}

type asyncHandler struct {
	UnimplementedHandler
}

func (h *asyncHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return &HandlerStartOperationResultAsync{
		OperationID: "async",
	}, nil
}

func TestAsync(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncHandler{})
	defer teardown()

	result, err := client.StartOperation(ctx, "foo", nil, StartOperationOptions{})
	require.NoError(t, err)
	require.NotNil(t, result.Pending)
}

type unsuccessfulHandler struct {
	UnimplementedHandler
}

func (h *unsuccessfulHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return nil, &UnsuccessfulOperationError{
		// We're passing the desired state via request ID in this test.
		State: OperationState(options.RequestID),
		Cause: fmt.Errorf("intentional"),
	}
}

func TestUnsuccessful(t *testing.T) {
	ctx, client, teardown := setup(t, &unsuccessfulHandler{})
	defer teardown()

	cases := []string{"canceled", "failed"}
	for _, c := range cases {
		_, err := client.StartOperation(ctx, "foo", nil, StartOperationOptions{RequestID: c})
		var unsuccessfulError *UnsuccessfulOperationError
		require.ErrorAs(t, err, &unsuccessfulError)
		require.Equal(t, OperationState(c), unsuccessfulError.State)
	}
}

type timeoutEchoHandler struct {
	UnimplementedHandler
}

func (h *timeoutEchoHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	deadline, set := ctx.Deadline()
	if !set {
		return &HandlerStartOperationResultSync[any]{
			Value: []byte("not set"),
		}, nil
	}
	return &HandlerStartOperationResultSync[any]{
		Value: []byte(formatDuration(time.Until(deadline))),
	}, nil
}

func TestStart_ContextDeadlinePropagated(t *testing.T) {
	ctx, client, teardown := setup(t, &timeoutEchoHandler{})
	defer teardown()

	deadline, _ := ctx.Deadline()
	initialTimeout := time.Until(deadline)
	result, err := client.StartOperation(ctx, "foo", nil, StartOperationOptions{})

	require.NoError(t, err)
	requireTimeoutPropagated(t, result, initialTimeout)
}

func TestStart_RequestTimeoutHeaderOverridesContextDeadline(t *testing.T) {
	// relies on ctx returned here having default testTimeout set greater than expected timeout
	ctx, client, teardown := setup(t, &timeoutEchoHandler{})
	defer teardown()

	timeout := 100 * time.Millisecond
	result, err := client.StartOperation(ctx, "foo", nil, StartOperationOptions{Header: Header{HeaderRequestTimeout: formatDuration(timeout)}})

	require.NoError(t, err)
	requireTimeoutPropagated(t, result, timeout)
}

func requireTimeoutPropagated(t *testing.T, result *ClientStartOperationResult[*LazyValue], expected time.Duration) {
	response := result.Successful
	require.NotNil(t, response)
	var responseBody []byte
	err := response.Consume(&responseBody)
	require.NoError(t, err)
	parsedTimeout, err := parseDuration(string(responseBody))
	require.NoError(t, err)
	require.NotZero(t, parsedTimeout)
	require.LessOrEqual(t, parsedTimeout, expected)
}

func TestStart_TimeoutNotPropagated(t *testing.T) {
	_, client, teardown := setup(t, &timeoutEchoHandler{})
	defer teardown()

	result, err := client.StartOperation(context.Background(), "foo", nil, StartOperationOptions{})

	require.NoError(t, err)
	response := result.Successful
	require.NotNil(t, response)
	var responseBody []byte
	err = response.Consume(&responseBody)
	require.NoError(t, err)
	require.Equal(t, []byte("not set"), responseBody)
}

func TestStart_NilContentHeaderDoesNotPanic(t *testing.T) {
	_, client, teardown := setup(t, &requestIDEchoHandler{})
	defer teardown()

	result, err := client.StartOperation(context.Background(), "op", &Content{Data: []byte("abc")}, StartOperationOptions{})

	require.NoError(t, err)
	response := result.Successful
	require.NotNil(t, response)
	var responseBody []byte
	err = response.Consume(&responseBody)
	require.NoError(t, err)
}
