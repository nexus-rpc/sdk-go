package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// ClientOptions are options for creating a Client.
type ClientOptions struct {
	// Base URL of the service.
	ServiceBaseURL string
	// A function for making HTTP requests.
	// Defaults to [http.DefaultClient.Do].
	HTTPCaller func(*http.Request) (*http.Response, error)
}

// User-Agent header set on HTTP requests.
const userAgent = "Nexus-go-sdk/" + version

const headerUserAgent = "User-Agent"

// Error indicating an empty ServiceBaseURL option was used to create a client when making a Nexus service request.
var errEmptyServiceBaseURL = errors.New("empty serviceBaseURL")

// Error indicating a non HTTP URL was used to create a [Client].
var errInvalidURLScheme = errors.New("invalid URL scheme")

var errEmptyOperationName = errors.New("empty operation name")

var errEmptyOperationID = errors.New("empty operation ID")

var errOperationWaitTimeout = errors.New("operation wait timeout")

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

// A Client makes Nexus service requests as defined in the [Nexus HTTP API].
//
// It can start a new operation and get an [OperationHandle] to an existing, asynchronous operation.
//
// Use an [OperationHandle] to cancel, get the result of, and get information about asynchronous operations.
//
// OperationHandles can be obtained either by starting new operations or by calling [Client.NewHandle] for existing
// operations.
//
// [Nexus HTTP API]: https://github.com/nexus-rpc/api
type Client struct {
	// The options this client was created with after applying defaults.
	options        ClientOptions
	serviceBaseURL *url.URL
}

// NewClient creates a new [Client] from provided [ClientOptions].
// Only BaseServiceURL is required.
func NewClient(options ClientOptions) (*Client, error) {
	if options.HTTPCaller == nil {
		options.HTTPCaller = http.DefaultClient.Do
	}
	if options.ServiceBaseURL == "" {
		return nil, errEmptyServiceBaseURL
	}
	var serviceBaseURL *url.URL
	var err error
	serviceBaseURL, err = url.Parse(options.ServiceBaseURL)
	if err != nil {
		return nil, err
	}
	if serviceBaseURL.Scheme != "http" && serviceBaseURL.Scheme != "https" {
		return nil, errInvalidURLScheme
	}

	return &Client{
		options:        options,
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
	if operation == "" {
		err = errEmptyOperationName
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
// One and only one of Successful or Pending will be non-nil.
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
//  3. The operation was unsuccessful. The returned result will be nil and error will be an
//     [UnsuccessfulOperationError].
//
//  4. Any other failure.
func (c *Client) StartOperation(ctx context.Context, options StartOperationOptions) (*StartOperationResult, error) {
	if closer, ok := options.Body.(io.Closer); ok {
		// Close the request body in case we error before sending the HTTP request (which may double close but that's fine since we ignore the error).
		defer closer.Close()
	}
	if options.Operation == "" {
		return nil, errEmptyOperationName
	}
	url := c.serviceBaseURL.JoinPath(url.PathEscape(options.Operation))

	if options.CallbackURL != "" {
		q := url.Query()
		q.Set(queryCallbackURL, options.CallbackURL)
		url.RawQuery = q.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, "POST", url.String(), options.Body)
	if err != nil {
		return nil, err
	}

	if options.Header != nil {
		request.Header = options.Header.Clone()
	}
	if options.RequestID == "" {
		requestIDFromHeader := options.Header.Get(headerRequestID)
		if requestIDFromHeader != "" {
			options.RequestID = requestIDFromHeader
		} else {
			options.RequestID = uuid.NewString()
		}
	}
	request.Header.Set(headerRequestID, options.RequestID)
	request.Header.Set(headerUserAgent, userAgent)

	response, err := c.options.HTTPCaller(request)
	if err != nil {
		return nil, err
	}
	// Do not close response body here to allow successful result to read it.
	if response.StatusCode == http.StatusOK {
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
				Operation: options.Operation,
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
	// Body of the operation request.
	// If it is an [io.Closer], the body is guaranteed to be closed in Client.ExecuteOperation.
	Body io.Reader
	// Header to attach to start and get-result HTTP requests. Optional.
	// Content-Type will be deleted in the get-result request.
	Header http.Header
	// Duration to wait for operation completion.
	//
	// ⚠ NOTE: unlike GetOperationResultOptions.Wait, zero and negative values are considered durations of MaxInt64.
	Wait time.Duration
}

// NewExecuteOperationOptions is shorthand for creating an [ExecuteOperationOptions] struct with a JSON body. Marshals
// the provided value to JSON using [json.Marshal] and sets the proper Content-Type header.
func NewExecuteOperationOptions(operation string, v any) (options ExecuteOperationOptions, err error) {
	if operation == "" {
		err = errEmptyOperationName
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

func (o *ExecuteOperationOptions) intoStartOptions() StartOperationOptions {
	return StartOperationOptions{
		Operation:   o.Operation,
		CallbackURL: o.CallbackURL,
		RequestID:   o.RequestID,
		Header:      o.Header,
		Body:        o.Body,
	}
}

func (o *ExecuteOperationOptions) intoGetResultOptions() (options GetOperationResultOptions) {
	options.Header = o.Header
	if options.Header != nil {
		options.Header = options.Header.Clone()
		options.Header.Del(headerContentType)
	}
	if o.Wait <= 0 {
		options.Wait = time.Duration(math.MaxInt64)
	} else {
		options.Wait = o.Wait
	}
	return
}

// ExecuteOperation is a helper for starting an operation and waiting for its completion.
//
// For asynchronous operations, the client will long poll for their result, issuing one or more requests until the
// wait period provided via [ExecuteOperationOptions] exceeds, in which case an [ErrOperationStillRunning] error is
// returned.
//
// The wait time is capped to the deadline of the provided context. Make sure to handle both context deadline errors and
// [ErrOperationStillRunning].
//
// Note that the wait period is enforced by the server and may not be respected if the server is misbehaving. Set the
// context deadline to the max allowed wait period to ensure this call returns in a timely fashion.
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
// Fails if provided an invalid operation or ID.
func (c *Client) NewHandle(operation string, operationID string) (*OperationHandle, error) {
	var es []error
	if operation == "" {
		es = append(es, errEmptyOperationName)
	}
	if operationID == "" {
		es = append(es, errEmptyOperationID)
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

func (c *Client) sendGetOperationRequest(ctx context.Context, request *http.Request) (*http.Response, error) {
	response, err := c.options.HTTPCaller(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusOK {
		return response, nil
	}

	// Do this once here and make sure it doesn't leak.
	body, err := readAndReplaceBody(response)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusRequestTimeout:
		return nil, errOperationWaitTimeout
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

// readAndReplaceBody reads the response body in its entirety and closes it, and then replaces the original response
// body with an in-memory buffer.
// The body is replaced even when there was an error reading the entire body.
func readAndReplaceBody(response *http.Response) ([]byte, error) {
	responseBody := response.Body
	body, err := io.ReadAll(responseBody)
	responseBody.Close()
	response.Body = io.NopCloser(bytes.NewReader(body))
	return body, err
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

func failureFromResponse(response *http.Response, body []byte) (Failure, error) {
	if !isContentTypeJSON(response.Header) {
		return Failure{}, newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get(headerContentType)), response, body)
	}
	var failure Failure
	err := json.Unmarshal(body, &failure)
	return failure, err
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
