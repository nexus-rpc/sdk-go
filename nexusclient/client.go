package nexusclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexusapi"
)

// Options for creating a Client.
type Options struct {
	// Base URL of the service.
	// Optional. If not provided, this client can only be used to deliver operation completions.
	ServiceBaseURL string
	// An HTTP Client to use for making HTTP requests.
	// Defaults to [http.DefaultClient].
	HTTPClient *http.Client
	// Max duration to wait for a single get result request.
	// Enforced if context deadline for the request is unset or greater than this value.
	//
	// Defaults to one minute.
	GetResultMaxRequestTimeout time.Duration
	// A stuctured logging handler.
	// Defaults to logging with text format to stderr at info level.
	LogHandler slog.Handler
	// Optional marshaler for marshaling objects to JSON.
	// Defaults to output with indentation.
	Marshaler nexusapi.Marshaler
}

// A Client is used to start an operation, get an [OperationHandle] to an existing operation, and deliver operation
// completions.
type Client struct {
	// The options this client was created with after applying defaults.
	Options        Options
	serviceBaseURL *url.URL
	logger         slog.Logger
}

var (
	// Package version.
	Version = "dev" // TODO: Actual version to be set by goreleaser
	// User agent used to make HTTP requests.
	UserAgent       = fmt.Sprintf("Nexus-go-sdk/%s", Version)
	headerUserAgent = "User-Agent"
	// Error indicating an empty ServiceBaseURL option was used to create a client when making a Nexus service request.
	ErrEmptyServiceBaseURL = errors.New("empty serviceBaseURL")
	// Error indicating a non HTTP URL was used to create a [Client].
	ErrInvalidURLScheme = errors.New("invalid URL scheme")

	// Asynchronous method was used on an [OperationHandle] representing a synchronous operation.
	ErrHandleForSyncOperation = errors.New("handle represents a synchronous operation")

	errOperationStillRunning = errors.New("operation still running")
)

// Error that indicates a client encountered something unexpected in the server's response.
type UnexpectedResponseError struct {
	// Error message.
	Message string
	// The HTTP response. The response body will have already been read into ResponseBody and closed.
	Response *http.Response
	// Body extracted from the HTTP response.
	ResponseBody []byte
}

// Error implements the error interface.
func (e *UnexpectedResponseError) Error() string {
	if nexusapi.IsContentTypeJSON(e.Response.Header) {
		var failure nexusapi.Failure
		if err := json.Unmarshal(e.ResponseBody, &failure); err == nil && failure.Message != "" {
			return fmt.Sprintf("%s: %s", e.Message, failure.Message)
		}

	}
	return e.Message
}

func (c *Client) newUnexpectedResponseError(message string, response *http.Response) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "error", err)
	}
	return &UnexpectedResponseError{
		Message:      message,
		Response:     response,
		ResponseBody: body,
	}
}

// NewClient creates a new [Client] from provided [Options].
// None of the options are required. Provide BaseServiceURL if you intend to use this client to make Nexus service calls
// or leave empty when using this client only to deliver completions.
func NewClient(options Options) (*Client, error) {
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	var serviceBaseURL *url.URL
	if options.ServiceBaseURL != "" {
		var err error
		serviceBaseURL, err = url.Parse(options.ServiceBaseURL)
		if err != nil {
			return nil, err
		}
		if serviceBaseURL.Scheme != "http" && serviceBaseURL.Scheme != "https" {
			return nil, ErrInvalidURLScheme
		}
	}
	if options.LogHandler == nil {
		options.LogHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	if options.GetResultMaxRequestTimeout == 0 {
		options.GetResultMaxRequestTimeout = time.Minute
	}
	if options.Marshaler == nil {
		options.Marshaler = nexusapi.DefaultMarshaler
	}

	return &Client{
		Options:        options,
		serviceBaseURL: serviceBaseURL,
		logger:         *slog.New(options.LogHandler),
	}, nil
}

// A Handle used to cancel operations and get their result and status.
// Must be explicitly closed.
type OperationHandle struct {
	operation string
	id        string
	state     nexusapi.OperationState

	client *Client

	// mutually exclusive with failure
	response *http.Response
	failure  *nexusapi.Failure
}

// Operation is the name of the operation this handle represents.
func (h *OperationHandle) Operation() string {
	return h.operation
}

// ID is the handler generated ID for this handle's operation.
// Empty for synchronous operations.
func (h *OperationHandle) ID() string {
	return h.id
}

// State is the last known operation state.
// Empty for handles created with [nexusclient.Client.GetHandle] before issuing a request to get the result.
//
// ⚠️ [nexusclient.OperationHandle.GetInfo] does not update the handle's State.
func (h *OperationHandle) State() nexusapi.OperationState {
	return h.state
}

// Close the handle's associated http response, if present.
func (h *OperationHandle) Close() error {
	// Body will have already been closed
	if h.state != nexusapi.OperationStateSucceeded {
		return nil
	}
	return h.response.Body.Close()
}

// Options for [nexusclient.OperationHandle.GetInfo].
type GetInfoOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
}

// GetHandle gets a handle to an asynchronous operation by name and ID.
// Does not incur a trip to the server.
func (c *Client) GetHandle(operation string, operationID string) *OperationHandle {
	return &OperationHandle{
		client:    c,
		operation: operation,
		id:        operationID,
	}
}

// GetInfo gets operation information issuing a network request to the service handler.
//
// ⚠️ Does not update the handle's State.
func (h *OperationHandle) GetInfo(ctx context.Context, options GetInfoOptions) (*nexusapi.OperationInfo, error) {
	if h.id == "" {
		return nil, ErrHandleForSyncOperation
	}
	return h.client.getOperationInfo(ctx, getOperationInfoRequest{
		Operation:   h.operation,
		OperationID: h.id,
		Header:      options.Header,
	})
}

type getOperationInfoRequest struct {
	Operation   string
	OperationID string
	Header      http.Header
}

func (c *Client) getOperationInfo(ctx context.Context, request getOperationInfoRequest) (*nexusapi.OperationInfo, error) {
	url, err := c.joinURL(request.Operation, request.OperationID)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if request.Header != nil {
		httpReq.Header = request.Header.Clone()
	}

	httpReq.Header.Set(headerUserAgent, UserAgent)
	response, err := c.Options.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, c.newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response)
	}

	return c.operationInfoFromResponse(response)
}

// Options for [nexusclient.OperationHandle.GetResult].
type GetResultOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
	// Boolean indicating whether to wait for operation completion or return the current status immediately.
	Wait bool
}

// GetResult gets the result of an operation.
//
// If the handle was obtained using the StartOperation API, and the operation completed synchronously, the response may
// already be stored in the handle, otherwise, the handle will use its associated client to issue a request to get the
// operation's result.
//
// By default, GetResult returns a nil response immediately and no error after issuing a call if the operation has not
// yet completed.
//
// Callers may set [nexusclient.GetResultOptions.Wait] to true to alter this behavior, causing the client to long poll
// for the result until the provided context deadline is exceeded. When the deadline exceeds, GetResult will return a
// nil response and [context.DeadlineExceeded] error. The client may issue multiple requests until the deadline exceeds
// with a max request timeout of [nexusclient.Options.GetResultMaxRequestTimeout].
func (h *OperationHandle) GetResult(ctx context.Context, options GetResultOptions) (*http.Response, error) {
	switch h.state {
	case nexusapi.OperationStateCanceled, nexusapi.OperationStateFailed:
		return nil, &nexusapi.UnsuccessfulOperationError{State: h.state, Failure: h.failure}
	case nexusapi.OperationStateSucceeded:
		return h.response, nil
	default:
		if h.id == "" {
			// This should never be possible as handles are constructed only by the client.
			panic("handle has no id")
		}
		var unsuccessfulError *nexusapi.UnsuccessfulOperationError
		response, err := h.client.getOperationResult(ctx, getOperationResultRequest{
			Operation:   h.operation,
			OperationID: h.id,
			Header:      options.Header,
			Wait:        options.Wait,
		})
		if err != nil {
			if errors.As(err, &unsuccessfulError) {
				h.state = unsuccessfulError.State
				h.failure = unsuccessfulError.Failure
			}
		} else if response != nil {
			h.response = response
			h.state = nexusapi.OperationStateSucceeded
		} else {
			h.state = nexusapi.OperationStateRunning
		}
		return response, err
	}
}

func (c *Client) getOperationResult(ctx context.Context, request getOperationResultRequest) (*http.Response, error) {
	url, err := c.joinURL(request.Operation, request.OperationID, "result")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	if request.Header != nil {
		httpReq.Header = request.Header.Clone()
	}
	httpReq.Header.Set(headerUserAgent, UserAgent)

	for {
		response, err := c.sendGetOperationResultRequest(ctx, httpReq, request.Wait)
		if err != nil {
			if errors.Is(err, errOperationStillRunning) {
				if request.Wait {
					continue
				} else {
					return nil, nil
				}
			}
			return nil, err
		}
		return response, nil
	}
}

func (c *Client) sendGetOperationResultRequest(ctx context.Context, httpReq *http.Request, wait bool) (*http.Response, error) {
	if wait {
		url := httpReq.URL
		timeout := c.Options.GetResultMaxRequestTimeout
		if deadline, set := ctx.Deadline(); set {
			timeout = time.Until(deadline)
			if timeout > c.Options.GetResultMaxRequestTimeout {
				timeout = c.Options.GetResultMaxRequestTimeout
			}
		}

		q := url.Query()
		q.Set(nexusapi.QueryWait, fmt.Sprintf("%dms", timeout.Milliseconds()))
		url.RawQuery = q.Encode()
	}

	response, err := c.Options.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == 200 {
		return response, nil
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 204:
		state := nexusapi.OperationState(response.Header.Get(nexusapi.HeaderOperationState))

		switch state {
		case nexusapi.OperationStateRunning:
			return nil, errOperationStillRunning
		case nexusapi.OperationStateSucceeded:
			return response, nil
		default:
			return nil, c.newUnexpectedResponseError(fmt.Sprintf("unexpected operation state: %s", state), response)
		}
	case nexusapi.StatusOperationFailed:
		state, err := c.getUnsuccessfulStateFromHeader(response)
		if err != nil {
			return nil, err
		}
		failure, err := c.failureFromResponse(response)
		if err != nil {
			return nil, err
		}
		return nil, &nexusapi.UnsuccessfulOperationError{
			State:   state,
			Failure: failure,
		}
	default:
		return nil, c.newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response)
	}
}

// Options for [nexusclient.OperationHandle.Cancel].
type CancelOptions struct {
	// Header to attach to the HTTP request. Optional.
	Header http.Header
}

// Cancel requests to cancel an asynchronous operation.
//
// Cancelation is asynchronous and may be not be respected by the operation's implementation.
func (h *OperationHandle) Cancel(ctx context.Context, options CancelOptions) error {
	if h.id == "" {
		return ErrHandleForSyncOperation
	}

	return h.client.cancelOperation(ctx, cancelOperationRequest{
		Operation:   h.operation,
		OperationID: h.id,
		Header:      options.Header,
	})
}

type cancelOperationRequest struct {
	Operation   string
	OperationID string
	Header      http.Header
}

func (c *Client) cancelOperation(ctx context.Context, request cancelOperationRequest) error {
	url, err := c.joinURL(request.Operation, request.OperationID, "cancel")
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url.String(), nil)
	if err != nil {
		return err
	}
	if request.Header != nil {
		httpReq.Header = request.Header.Clone()
	}

	httpReq.Header.Set(headerUserAgent, UserAgent)
	response, err := c.Options.HTTPClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return c.newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response)
	}
	return nil
}

// Input for [nexusclient.Client.StartOperation].
type StartOperationRequest struct {
	// Name of the operation to start.
	Operation string
	// Callback URL to provide to the handle for receiving async operation completions. Optional.
	CallbackURL string
	// Request ID that may be used by the server handler to dedupe this start request.
	// By default a v4 UUID will be generated by the client.
	RequestID string
	// Header to attach to the HTTP request. Optional.
	Header http.Header
	// Body of the operation request.
	// If it is an [io.Closer], the body will be automatically closed by the client.
	Body io.Reader
}

// NewJSONStartOperationRequest is shorthand for creating a StartOperationRequest with a JSON body.
// Marhsals the provided value to JSON using [nexusapi.DefaultMarshaler].
func NewJSONStartOperationRequest(operation string, v any) (request StartOperationRequest, err error) {
	req := StartOperationRequest{}
	var b []byte
	b, err = nexusapi.DefaultMarshaler(v)
	if err != nil {
		return
	}
	req.Operation = operation
	req.Header = http.Header{nexusapi.HeaderContentType: []string{nexusapi.ContentTypeJSON}}
	req.Body = bytes.NewReader(b)
	return
}

// StartOperation calls the configured Nexus endpoint to start an operation.
// The operation may complete synchronously, delivering the result inline, or asynchronously returning a reference that
// can be used via the returned handle to interact with the operation.
//
// Use the returned [OperationHandle] to retrieve the operation's result.
func (c *Client) StartOperation(ctx context.Context, request StartOperationRequest) (*OperationHandle, error) {
	if closer, ok := request.Body.(io.Closer); ok {
		// Close the request body in case we error before sending the HTTP request.
		defer closer.Close()
	}
	url, err := c.joinURL(request.Operation)
	if err != nil {
		return nil, err
	}
	if request.CallbackURL != "" {
		q := url.Query()
		q.Set(nexusapi.QueryCallbackURL, request.CallbackURL)
		url.RawQuery = q.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url.String(), request.Body)
	if err != nil {
		return nil, err
	}

	if request.Header != nil {
		httpReq.Header = request.Header.Clone()
	}
	if request.RequestID == "" {
		requestIDFromHeader := request.Header.Get(nexusapi.HeaderRequestID)
		if requestIDFromHeader != "" {
			request.RequestID = requestIDFromHeader
		} else {
			request.RequestID = uuid.NewString()
		}
	}
	httpReq.Header.Set(nexusapi.HeaderRequestID, request.RequestID)

	httpReq.Header.Set(headerUserAgent, UserAgent)
	response, err := c.Options.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	// Do not close response body here to allow successful result to read it.
	switch response.StatusCode {
	case 200, 204:
		handle := &OperationHandle{
			operation: request.Operation,
			state:     nexusapi.OperationStateSucceeded,
			response:  response,
		}
		return handle, nil
	}

	// Do this once here and make sure it doesn't leak.
	defer response.Body.Close()

	switch response.StatusCode {
	case 201:
		info, err := c.operationInfoFromResponse(response)
		if err != nil {
			return nil, err
		}
		if info.State != nexusapi.OperationStateRunning {
			return nil, c.newUnexpectedResponseError(fmt.Sprintf("invalid operation state in response info: %q", info.State), response)
		}
		handle := &OperationHandle{
			client:    c,
			operation: request.Operation,
			id:        info.ID,
			state:     info.State,
			response:  response,
		}
		return handle, nil
	case nexusapi.StatusOperationFailed:
		state, err := c.getUnsuccessfulStateFromHeader(response)
		if err != nil {
			return nil, err
		}

		failure, err := c.failureFromResponse(response)
		if err != nil {
			return nil, err
		}

		return &OperationHandle{
			operation: request.Operation,
			state:     state,
			failure:   failure,
		}, nil
	default:
		return nil, c.newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response)
	}
}

type getOperationResultRequest struct {
	Operation   string
	OperationID string
	Wait        bool
	Header      http.Header
}

func (c *Client) operationInfoFromResponse(response *http.Response) (*nexusapi.OperationInfo, error) {
	if !nexusapi.IsContentTypeJSON(response.Header) {
		return nil, c.newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get(nexusapi.HeaderContentType)), response)
	}
	var info nexusapi.OperationInfo
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) failureFromResponse(response *http.Response) (*nexusapi.Failure, error) {
	if !nexusapi.IsContentTypeJSON(response.Header) {
		return nil, c.newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get(nexusapi.HeaderContentType)), response)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var failure nexusapi.Failure
	if err := json.Unmarshal(body, &failure); err != nil {
		return nil, err
	}
	return &failure, nil
}

func (c *Client) getUnsuccessfulStateFromHeader(response *http.Response) (nexusapi.OperationState, error) {
	state := nexusapi.OperationState(response.Header.Get(nexusapi.HeaderOperationState))
	switch state {
	case nexusapi.OperationStateCanceled:
		return state, nil
	case nexusapi.OperationStateFailed:
		return state, nil
	default:
		return state, c.newUnexpectedResponseError(fmt.Sprintf("invalid operation state header: %q", state), response)
	}
}

// Input for [nexusclient.Client.CompletionOperation].
// It has two implementations: [OperationCompletionSuccessful] and [OperationCompletionUnsuccessful].
type OperationCompletion interface {
	io.Closer
	applyToHTTPRequest(*http.Request, *Client) error
}

// Input for [nexusclient.Client.CompletionOperation] to deliver successful operation results.
type OperationCompletionSuccessful struct {
	// Header to send in the completion request.
	Header http.Header
	// Body to send in the completion HTTP request.
	// If it implements `io.Closer` it will automatically be closed by the client
	Body io.Reader
}

// Input for [nexusclient.Client.CompletionOperation] to deliver unsuccessful operation results.
type OperationCompletionUnsuccessful struct {
	// Header to send in the completion request.
	Header http.Header
	// State of the operation, should be failed or canceled.
	State nexusapi.OperationState
	// Failure object to send with the completion.
	Failure *nexusapi.Failure
}

// NewJSONSuccessfulOperationCompletion constructs an [OperationCompletionSuccessful] from a JSONable value.
// Marhsals the provided value to JSON using [nexusapi.DefaultMarshaler].
func NewJSONSuccessfulOperationCompletion(v any) (*OperationCompletionSuccessful, error) {
	b, err := nexusapi.DefaultMarshaler(v)
	if err != nil {
		return nil, err
	}

	header := make(http.Header, 1)
	header.Set(nexusapi.HeaderContentType, nexusapi.ContentTypeJSON)

	return &OperationCompletionSuccessful{
		Header: header,
		Body:   io.NopCloser(bytes.NewReader(b)),
	}, nil
}

func (c *OperationCompletionSuccessful) applyToHTTPRequest(request *http.Request, client *Client) error {
	if c.Header != nil {
		request.Header = c.Header.Clone()
	}
	request.Header.Set(nexusapi.HeaderOperationState, string(nexusapi.OperationStateSucceeded))
	if closer, ok := c.Body.(io.ReadCloser); ok {
		request.Body = closer
	} else {
		request.Body = io.NopCloser(c.Body)
	}
	return nil
}

// Close implements the io.Closer interface.
func (c *OperationCompletionSuccessful) Close() error {
	if closer, ok := c.Body.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (c *OperationCompletionUnsuccessful) applyToHTTPRequest(request *http.Request, client *Client) error {
	if c.Header != nil {
		request.Header = c.Header.Clone()
	}
	request.Header.Set(nexusapi.HeaderOperationState, string(c.State))
	request.Header.Set(nexusapi.HeaderContentType, nexusapi.ContentTypeJSON)

	b, err := client.Options.Marshaler(c.Failure)
	if err != nil {
		return err
	}

	request.Body = io.NopCloser(bytes.NewReader(b))
	return nil
}

// Close implements the io.Closer interface.
func (c *OperationCompletionUnsuccessful) Close() error {
	return nil
}

// DeliverCompletion delivers the result of a completed asynchronous operation to the provided URL.
func (c *Client) DeliverCompletion(ctx context.Context, url string, completion OperationCompletion) error {
	// while the http client is expected to close the body, we close in case request creation fails
	defer completion.Close()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}
	if err := completion.applyToHTTPRequest(httpReq, c); err != nil {
		return err
	}

	httpReq.Header.Set(headerUserAgent, UserAgent)
	response, err := c.Options.HTTPClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return c.newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response)
	}

	return nil
}

func (c *Client) joinURL(parts ...string) (*url.URL, error) {
	if c.serviceBaseURL == nil {
		return nil, ErrEmptyServiceBaseURL
	}
	return c.serviceBaseURL.JoinPath(parts...), nil
}
