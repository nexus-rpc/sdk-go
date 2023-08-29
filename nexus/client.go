package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// ClientOptions are options for creating a Client.
type ClientOptions struct {
	// Base URL of the service.
	// Optional. If not provided, created clients can only be used to deliver operation completions.
	ServiceBaseURL string
	// A function for making HTTP requests.
	// Defaults to [http.DefaultClient.Do].
	HTTPCaller func(*http.Request) (*http.Response, error)
	// Max duration to wait for a single request to get an operation's result.
	// Enforced if wait time provided via the various method options is unset or greater than this value.
	//
	// Defaults to one minute.
	GetResultMaxTimeout time.Duration
	// Optional marshaler for marshaling objects to JSON.
	// Defaults to json.Marshal.
	Marshaler func(any) ([]byte, error)
}

// User-Agent header set on HTTP requests.
const userAgent = "Nexus-go-sdk/" + Version

const headerUserAgent = "User-Agent"

const infiniteDuration time.Duration = 9223372036854775807

// Error indicating an empty ServiceBaseURL option was used to create a client when making a Nexus service request.
var errEmptyServiceBaseURL = errors.New("empty serviceBaseURL")

// Error indicating a non HTTP URL was used to create a [Client].
var errInvalidURLScheme = errors.New("invalid URL scheme")

var errInvalidOperationName = errors.New("invalid operation name")

var errInvalidOperationID = errors.New("invalid operation ID")

// ErrOperationStillRunning indicates that an operation is still running while trying to get its result.
var ErrOperationStillRunning = errors.New("operation still running")

// Error that indicates a client encountered something unexpected in the server's response.
type UnexpectedResponseError struct {
	// Error message.
	Message string
	// The HTTP response. The response body will have already been read into memory and does not need to be closed.
	Response *http.Response
	// Optional failure that may have been emedded in the HTTP response body.
	Failure *Failure
}

// Error implements the error interface.
func (e *UnexpectedResponseError) Error() string {
	return e.Message
}

func newUnexpectedResponseError(message string, response *http.Response, body []byte) error {
	var failure *Failure
	if isContentTypeJSON(response.Header) {
		if err := json.Unmarshal(body, &failure); err == nil && failure.Message != "" {
			message += ": " + failure.Message
		}
	}

	return &UnexpectedResponseError{
		Message:  message,
		Response: response,
		Failure:  failure,
	}
}

// A Client is makes Nexus service requests  as defined in the [Nexus HTTP API].
//
// It can start an operation, get an [OperationHandle] to an existing operation, and deliver operation completions.
//
// Use an [OperationHandle] to cancel, get the result of, and get information about asynchronous operations.
//
// OperationHandles can be obtained either by starting new operations or by calling [Client.NewHandle] for existing
// operations.
//
// [Nexus HTTP API]: https://github.com/nexus-rpc/api
type Client struct {
	// The options this client was created with after applying defaults.
	Options        ClientOptions
	serviceBaseURL *url.URL
}

// NewClient creates a new [Client] from provided [ClientOptions].
// None of the options are required. Provide BaseServiceURL if you intend to use this client to make Nexus service calls
// or leave empty when using this client only to deliver completions.
func NewClient(options ClientOptions) (*Client, error) {
	if options.HTTPCaller == nil {
		options.HTTPCaller = http.DefaultClient.Do
	}
	var serviceBaseURL *url.URL
	if options.ServiceBaseURL != "" {
		var err error
		serviceBaseURL, err = url.Parse(options.ServiceBaseURL)
		if err != nil {
			return nil, err
		}
		if serviceBaseURL.Scheme != "http" && serviceBaseURL.Scheme != "https" {
			return nil, errInvalidURLScheme
		}
	}
	if options.GetResultMaxTimeout == 0 {
		options.GetResultMaxTimeout = time.Minute
	}
	if options.Marshaler == nil {
		options.Marshaler = json.Marshal
	}

	return &Client{
		Options:        options,
		serviceBaseURL: serviceBaseURL,
	}, nil
}

// StartOperationOptions is input for [Client.StartOperation].
type StartOperationOptions struct {
	// Name of the operation to start.
	Operation string
	// Callback URL to provide to the handle for receiving async operation completions. Optional.
	// Implement a [CompletionHandler] and expose it as an HTTP handler to handle async completions.
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

// NewStartOperationOptions is shorthand for creating a [StartOperationOptions] struct with a JSON body. Marshals the
// provided value to JSON using [json.Marshal] and sets the proper Content-Type header.
func NewStartOperationOptions(operation string, v any) (options StartOperationOptions, err error) {
	if !isValidOperationName.MatchString(operation) {
		err = errInvalidOperationName
		return
	}
	var b []byte
	b, err = json.Marshal(v)
	if err != nil {
		return
	}
	options.Operation = operation
	options.Header = http.Header{headerContentType: []string{contentTypeJSON}}
	options.Body = bytes.NewReader(b)
	return
}

// StartOperationResult is the return value of [Client.StartOperation].
// It represents the mutually exclusive Successful and Pending outcomes of that method.
type StartOperationResult struct {
	// Set when start completes synchronously and successfully.
	//
	// ⚠️ The response body must be read in its entirety and closed to free up the underlying connection.
	Successful *http.Response
	// Set when the handler indicates that it started an asynchronous operation.
	// The attached handle can be used to perform actions such as cancel the operation or get its result.
	Pending *OperationHandle
}

// StartOperation calls the configured Nexus endpoint to start an operation.
//
// This method has the following possible outcomes:
//
//  1. The operation completes successfully. The HTTP response of this call will be set as
//     StartOperationResult.Successful and its body must be read in its entirety and closed to free up the underlying
//     connection.
//
//  2. The operation was started and the handler has indicated that it will complete asynchronously. An
//     [OperationHandle] will be returned as StartOperationResult.Pending, which can be used to perform actions such
//     as getting its result.
//
//  3. The operation completes unsuccessfully. The returned result will be nil and error will be an
//     [UnsuccessfulOperationError].
//
//  4. Any other failure.
//
// Example:
//
//	options, _ := NewStartOperationOptions("example", MyStruct{Field: "value"})
//	result, err := client.StartOperation(ctx, options)
//	if err != nil {
//		var unsuccessfulOperationError *UnsuccessfulOperationError
//		if errors.As(err, &unsuccessfulOperationError) { // operation failed or canceled
//			fmt.Printf("Operation unsuccessful with state: %s, failure message: %s\n", err.State, err.Failure.Message)
//		}
//		return err
//	}
//	if result.Successful { // operation successful
//		response := result.Successful
//		defer response.Body.Close()
//		fmt.Printf("Got response with content type: %s\n", response.Header.Get("Content-Type"))
//		body, _ := io.ReadAll(response.Body)
//	} else { // operation started asynchronously
//		handle := result.Pending
//		fmt.Printf("Started asynchronous operation with ID: %s\n", handle.ID)
//	}
func (c *Client) StartOperation(ctx context.Context, request StartOperationOptions) (*StartOperationResult, error) {
	if closer, ok := request.Body.(io.Closer); ok {
		// Close the request body in case we error before sending the HTTP request (which may double close but that's fine since we ignore the error).
		defer closer.Close()
	}
	if !isValidOperationName.MatchString(request.Operation) {
		return nil, errInvalidOperationName
	}
	url, err := c.joinURL(request.Operation)
	if err != nil {
		return nil, err
	}
	if request.CallbackURL != "" {
		q := url.Query()
		q.Set(queryCallbackURL, request.CallbackURL)
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
		requestIDFromHeader := request.Header.Get(headerRequestID)
		if requestIDFromHeader != "" {
			request.RequestID = requestIDFromHeader
		} else {
			request.RequestID = uuid.NewString()
		}
	}
	httpReq.Header.Set(headerRequestID, request.RequestID)
	httpReq.Header.Set(headerUserAgent, userAgent)

	response, err := c.Options.HTTPCaller(httpReq)
	if err != nil {
		return nil, err
	}
	// Do not close response body here to allow successful result to read it.
	if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusNoContent {
		return &StartOperationResult{
			Successful: response,
		}, nil
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusCreated:
		info, err := operationInfoFromResponse(response, body)
		if err != nil {
			return nil, err
		}
		if info.State != OperationStateRunning {
			return nil, newUnexpectedResponseError(fmt.Sprintf("invalid operation state in response info: %q", info.State), response, body)
		}
		return &StartOperationResult{
			Pending: &OperationHandle{
				Operation: request.Operation,
				ID:        info.ID,
				client:    c,
			},
		}, nil
	case statusOperationFailed:
		state, err := getUnsuccessfulStateFromHeader(response, body)
		if err != nil {
			return nil, err
		}

		failure, err := failureFromResponse(response, body)
		if err != nil {
			return nil, err
		}

		return nil, &UnsuccessfulOperationError{
			State:   state,
			Failure: failure,
		}
	default:
		return nil, newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response, body)
	}
}

// ExecuteOperationOptions is input for [Client.ExecuteOperation].
type ExecuteOperationOptions struct {
	// Name of the operation to start.
	Operation string
	// Callback URL to provide to the handle for receiving async operation completions. Optional.
	// Even though Client.ExecuteOperation waits for operation completion, some application may want to set this
	// callback as a fallback mechanism.
	CallbackURL string
	// Request ID that may be used by the server handler to dedupe this start request.
	// By default a v4 UUID will be generated by the client.
	RequestID string
	// Header to attach to the start HTTP request. Optional.
	StartHeader http.Header
	// Body of the operation request.
	// If it is an [io.Closer], the body is guaranteed to be closed in Client.ExecuteOperation.
	Body io.Reader
	// Header to attach to the get-result HTTP request. Optional.
	GetResultHeader http.Header
	// Duration to wait for operation completion.
	//
	// ⚠ NOTE: unlike GetResultOptions.Wait, zero and negative values imply infinite wait.
	Wait time.Duration
}

// NewExecuteOperationOptions is shorthand for creating an [ExecuteOperationOptions] struct with a JSON body. Marshals
// the provided value to JSON using [json.Marshal] and sets the proper Content-Type header.
func NewExecuteOperationOptions(operation string, v any) (options ExecuteOperationOptions, err error) {
	if !isValidOperationName.MatchString(operation) {
		err = errInvalidOperationName
		return
	}
	var b []byte
	b, err = json.Marshal(v)
	if err != nil {
		return
	}
	options.Operation = operation
	options.StartHeader = http.Header{headerContentType: []string{contentTypeJSON}}
	options.Body = bytes.NewReader(b)
	return
}

func (r *ExecuteOperationOptions) intoStartOptions() StartOperationOptions {
	return StartOperationOptions{
		Operation:   r.Operation,
		CallbackURL: r.CallbackURL,
		RequestID:   r.RequestID,
		Header:      r.StartHeader,
		Body:        r.Body,
	}
}

func (r *ExecuteOperationOptions) intoGetResultOptions() (options GetResultOptions) {
	options.Header = r.GetResultHeader
	if r.Wait <= 0 {
		options.Wait = infiniteDuration
	} else {
		options.Wait = r.Wait
	}
	return
}

// ExecuteOperation is a helper for starting an operation and waiting for its completion.
//
// For asynchronous operations, the client will long poll for their result, issuing one or more requests until the
// wait period provided via [ExecuteOperationOptions] exceeds, in which case an [ErrOperationStillRunning] error is
// returned.
//
// The wait time is capped to the deadline of the provided context.
//
// ⚠️ If this method completes successfully, the returned response's body must be read in its entirety and closed to
// free up the underlying connection.
func (c *Client) ExecuteOperation(ctx context.Context, request ExecuteOperationOptions) (*http.Response, error) {
	result, err := c.StartOperation(ctx, request.intoStartOptions())
	if err != nil {
		return nil, err
	}
	if result.Successful != nil {
		return result.Successful, nil
	}
	handle := result.Pending
	return handle.GetResult(ctx, request.intoGetResultOptions())
}

// NewHandle gets a handle to an asynchronous operation by name and ID.
// Does not incur a trip to the server.
// Fails if provided any empty operation or ID, or the client has no associated ServiceBaseURL.
func (c *Client) NewHandle(operation string, operationID string) (*OperationHandle, error) {
	var es []error
	if c.serviceBaseURL == nil {
		es = append(es, errEmptyServiceBaseURL)
	}
	if !isValidOperationName.MatchString(operation) {
		es = append(es, errInvalidOperationName)
	}
	if !isValidOperationID.MatchString(operationID) {
		es = append(es, errInvalidOperationID)
	}
	if len(es) > 0 {
		return nil, errors.Join(es...)
	}
	return &OperationHandle{
		client:    c,
		Operation: operation,
		ID:        operationID,
	}, nil
}

func (c *Client) sendGetOperationResultRequest(ctx context.Context, httpReq *http.Request, wait time.Duration) (*http.Response, error) {
	if wait > 0 {
		timeout := min(wait, c.Options.GetResultMaxTimeout)
		if deadline, set := ctx.Deadline(); set {
			timeout = min(timeout, time.Until(deadline))
		}

		url := httpReq.URL
		q := url.Query()
		q.Set(queryWait, fmt.Sprintf("%dms", timeout.Milliseconds()))
		url.RawQuery = q.Encode()
	}

	response, err := c.Options.HTTPCaller(httpReq)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusNoContent {
		return response, nil
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case statusOperationRunning:
		return nil, ErrOperationStillRunning
	case statusOperationFailed:
		state, err := getUnsuccessfulStateFromHeader(response, body)
		if err != nil {
			return nil, err
		}
		failure, err := failureFromResponse(response, body)
		if err != nil {
			return nil, err
		}
		return nil, &UnsuccessfulOperationError{
			State:   state,
			Failure: failure,
		}
	default:
		return nil, newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response, body)
	}
}

// DeliverCompletion delivers the result of a completed asynchronous operation to the provided URL.
// If completion is an [OperationCompletionSuccessful] its body will be automatically closed.
func (c *Client) DeliverCompletion(ctx context.Context, url string, completion OperationCompletion) error {
	// while the http client is expected to close the body, we close in case request creation fails (which may double close but that's fine since we ignore the error).
	defer completion.Close()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}
	if err := completion.applyToHTTPRequest(httpReq, c); err != nil {
		return err
	}

	httpReq.Header.Set(headerUserAgent, userAgent)
	response, err := c.Options.HTTPCaller(httpReq)
	if err != nil {
		return err
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return newUnexpectedResponseError(fmt.Sprintf("unexpected response status: %q", response.Status), response, body)
	}

	return nil
}

func (c *Client) joinURL(parts ...string) (*url.URL, error) {
	if c.serviceBaseURL == nil {
		return nil, errEmptyServiceBaseURL
	}
	return c.serviceBaseURL.JoinPath(parts...), nil
}

// OperationCompletion is input for [Client.DeliverCompletion].
// It has two implementations: [OperationCompletionSuccessful] and [OperationCompletionUnsuccessful].
type OperationCompletion interface {
	io.Closer
	applyToHTTPRequest(*http.Request, *Client) error
}

// OperationCompletionSuccessful is input for [Client.DeliverCompletion], used to deliver successful operation results.
type OperationCompletionSuccessful struct {
	// Header to send in the completion request.
	Header http.Header
	// Body to send in the completion HTTP request.
	// If it implements `io.Closer` it will automatically be closed by the client.
	Body io.Reader
}

// NewSuccessfulOperationCompletion constructs an [OperationCompletionSuccessful] from a JSONable value.
// Marshals the provided value to JSON using [json.Marshal] and sets the proper Content-Type header.
func NewSuccessfulOperationCompletion(v any) (*OperationCompletionSuccessful, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	header := make(http.Header, 1)
	header.Set(headerContentType, contentTypeJSON)

	return &OperationCompletionSuccessful{
		Header: header,
		Body:   io.NopCloser(bytes.NewReader(b)),
	}, nil
}

// OperationCompletionUnsuccessful is input for [Client.DeliverCompletion], used to deliver unsuccessful operation
// results.
type OperationCompletionUnsuccessful struct {
	// Header to send in the completion request.
	Header http.Header
	// State of the operation, should be failed or canceled.
	State OperationState
	// Failure object to send with the completion.
	Failure *Failure
}

func (c *OperationCompletionSuccessful) applyToHTTPRequest(request *http.Request, client *Client) error {
	if c.Header != nil {
		request.Header = c.Header.Clone()
	}
	request.Header.Set(headerOperationState, string(OperationStateSucceeded))
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
	request.Header.Set(headerOperationState, string(c.State))
	request.Header.Set(headerContentType, contentTypeJSON)

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

// readAndReplaceBody reads the response body in its entirety and closes it, and then replaces the original response
// body with an in-memory buffer.
func readAndReplaceBody(response *http.Response) ([]byte, error) {
	responseBody := response.Body
	defer responseBody.Close()
	body, err := io.ReadAll(responseBody)
	if err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func operationInfoFromResponse(response *http.Response, body []byte) (*OperationInfo, error) {
	if !isContentTypeJSON(response.Header) {
		return nil, newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get(headerContentType)), response, body)
	}
	var info OperationInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func failureFromResponse(response *http.Response, body []byte) (*Failure, error) {
	if !isContentTypeJSON(response.Header) {
		return nil, newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get(headerContentType)), response, body)
	}
	var failure Failure
	if err := json.Unmarshal(body, &failure); err != nil {
		return nil, err
	}
	return &failure, nil
}

func getUnsuccessfulStateFromHeader(response *http.Response, body []byte) (OperationState, error) {
	state := OperationState(response.Header.Get(headerOperationState))
	switch state {
	case OperationStateCanceled:
		return state, nil
	case OperationStateFailed:
		return state, nil
	default:
		return state, newUnexpectedResponseError(fmt.Sprintf("invalid operation state header: %q", state), response, body)
	}
}
