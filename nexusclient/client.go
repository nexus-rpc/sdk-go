package nexusclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexusapi"
	"golang.org/x/exp/slog"
)

type Options struct {
	// Base URL of the service.
	// Optional, if not provided, this client can only be used to deliver operation completions.
	ServiceBaseURL string
	// An Client to use for making HTTP requests.
	// Defaults to http.DefaultClient.
	HTTPClient *http.Client
	// Max duration to wait for a single get result request.
	// Enforced if context deadline for the request is unset or greater than this value.
	// Defaults to one minute.
	GetResultMaxRequestTimeout time.Duration
	// A stuctured logging handler.
	// Defaults to logging with text format to stderr at info level.
	LogHandler slog.Handler
	// Optional marshaler for marshaling objects to JSON.
	// Defaults to output with indentation.
	Marshaler nexusapi.Marshaler
}

type Client struct {
	// The options this client was created with after applying defaults.
	Options        Options
	serviceBaseURL *url.URL
	logger         slog.Logger
}

var (
	ErrEmptyServiceBaseURL = errors.New("empty serviceBaseURL")
	ErrInvalidURLScheme    = errors.New("invalid URL scheme")
)

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
	// TODO: default user agent (not here, in all requests constructed)

	return &Client{
		Options:        options,
		serviceBaseURL: serviceBaseURL,
		logger:         *slog.New(options.LogHandler),
	}, nil
}

type OperationHandle struct {
	operation string
	id        string

	client *Client
	state  nexusapi.OperationState

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

// TODO: note that GetInfo does not update the state
func (h *OperationHandle) State() nexusapi.OperationState {
	return h.state
}

func (h *OperationHandle) Close() error {
	// Body will have already been closed
	if h.state != nexusapi.OperationStateSucceeded {
		return nil
	}
	return h.response.Body.Close()
}

var ErrHandleForSyncOperation = errors.New("handle represents a synchronous operation")

type GetInfoOptions struct {
	Header http.Header
}

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

type GetResultOptions struct {
	Header http.Header
	Wait   bool
}

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
			}
		} else if response != nil {
			h.state = nexusapi.OperationStateSucceeded
		} else {
			h.state = nexusapi.OperationStateRunning
		}
		return response, err
	}
}

type CancelOptions struct {
	Header http.Header
}

func (h *OperationHandle) Cancel(ctx context.Context, options CancelOptions) error {
	return h.client.cancelOperation(ctx, cancelOperationRequest{
		Operation:   h.operation,
		OperationID: h.id,
		Header:      options.Header,
	})
}

type UnexpectedResponseError struct {
	Message      string
	Response     *http.Response
	ResponseBody []byte
}

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

func (c *Client) joinURL(parts ...string) (*url.URL, error) {
	if c.serviceBaseURL == nil {
		return nil, ErrEmptyServiceBaseURL
	}
	return c.serviceBaseURL.JoinPath(parts...), nil
}

func (c *Client) GetHandle(operation string, operationID string) *OperationHandle {
	return &OperationHandle{
		client:    c,
		operation: operation,
		id:        operationID,
	}
}

type StartOperationRequest struct {
	Operation   string
	CallbackURL string
	RequestID   string
	Header      http.Header
	Body        io.Reader
}

func (c *Client) StartOperation(ctx context.Context, request StartOperationRequest) (*OperationHandle, error) {
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
	httpReq.Header = request.Header.Clone()

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

type getOperationResultRequest struct {
	Operation   string
	OperationID string
	Wait        bool
	Header      http.Header
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
	httpReq.Header = request.Header.Clone()

	for {
		response, err := c.getOperationResultOnce(ctx, httpReq, request.Wait)
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

var errOperationStillRunning = errors.New("operation still running")

func (c *Client) getOperationResultOnce(ctx context.Context, httpReq *http.Request, wait bool) (*http.Response, error) {
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
	httpReq.Header = request.Header.Clone()

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

type OperationCompletion interface {
	io.Closer
	applyToHTTPRequest(*http.Request, *Client) error
}

type SuccessfulOperationCompletion struct {
	Header http.Header
	Body   io.ReadCloser
}

type UnsuccessfulOperationCompletion struct {
	State   nexusapi.OperationState
	Header  http.Header
	Failure *nexusapi.Failure
}

func NewBytesSuccessfulOperationCompletion(header http.Header, b []byte) *SuccessfulOperationCompletion {
	return &SuccessfulOperationCompletion{
		Header: header,
		Body:   io.NopCloser(bytes.NewReader(b)),
	}
}

func NewJSONSuccessfulOperationCompletion(header http.Header, v any) (*SuccessfulOperationCompletion, error) {
	b, err := nexusapi.DefaultMarshaler(v)
	if err != nil {
		return nil, err
	}

	return &SuccessfulOperationCompletion{
		Header: header,
		Body:   io.NopCloser(bytes.NewReader(b)),
	}, nil
}

func (c *SuccessfulOperationCompletion) applyToHTTPRequest(request *http.Request, client *Client) error {
	if c.Header != nil {
		request.Header = c.Header.Clone()
	}
	request.Header.Set(nexusapi.HeaderOperationState, string(nexusapi.OperationStateSucceeded))
	request.Body = c.Body
	return nil
}

func (c *SuccessfulOperationCompletion) Close() error {
	return c.Body.Close()
}

func (c *UnsuccessfulOperationCompletion) applyToHTTPRequest(request *http.Request, client *Client) error {
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

func (c *UnsuccessfulOperationCompletion) Close() error {
	return nil
}

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
