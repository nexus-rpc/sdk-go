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
	// A [Serializer] to customize client serialization behavior.
	// By default the client handles, JSONables, byte slices, and nil.
	Serializer Serializer
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
	if isMediaTypeJSON(response.Header.Get("Content-Type")) {
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
	if options.Serializer == nil {
		options.Serializer = defaultSerializer
	}

	return &Client{
		options:        options,
		serviceBaseURL: serviceBaseURL,
	}, nil
}

// ClientStartOperationResult is the return type of [Client.StartOperation].
// One and only one of Successful or Pending will be non-nil.
type ClientStartOperationResult[T any] struct {
	// Set when start completes synchronously and successfully.
	//
	// If T is a [LazyValue], ensure that your consume it or read the underlying content in its entirety and close it to
	// free up the underlying connection.
	Successful T
	// Set when the handler indicates that it started an asynchronous operation.
	// The attached handle can be used to perform actions such as cancel the operation or get its result.
	Pending *OperationHandle[T]
}

// StartOperation calls the configured Nexus endpoint to start an operation.
//
// This method has the following possible outcomes:
//
//  1. The operation completes successfully. The result of this call will be set as a [LazyValue] in
//     ClientStartOperationResult.Successful and must be consumed to free up the underlying connection.
//
//  2. The operation was started and the handler has indicated that it will complete asynchronously. An
//     [OperationHandle] will be returned as ClientStartOperationResult.Pending, which can be used to perform actions
//     such as getting its result.
//
//  3. The operation was unsuccessful. The returned result will be nil and error will be an
//     [UnsuccessfulOperationError].
//
//  4. Any other error.
func (c *Client) StartOperation(ctx context.Context, operation string, input any, options StartOperationOptions) (*ClientStartOperationResult[*LazyValue], error) {
	var reader *Reader
	if r, ok := input.(*Reader); ok {
		// Close the input reader in case we error before sending the HTTP request (which may double close but
		// that's fine since we ignore the error).
		defer r.Reader.Close()
		reader = r
	} else {
		content, ok := input.(*Content)
		if !ok {
			var err error
			content, err = c.options.Serializer.Serialize(input)
			if err != nil {
				return nil, err
			}
		}

		reader = &Reader{
			Header: content.Header,
			Reader: io.NopCloser(bytes.NewReader(content.Data)),
		}
	}

	url := c.serviceBaseURL.JoinPath(url.PathEscape(operation))

	if options.CallbackURL != "" {
		q := url.Query()
		q.Set(queryCallbackURL, options.CallbackURL)
		url.RawQuery = q.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, "POST", url.String(), reader.Reader)
	if err != nil {
		return nil, err
	}

	// Use provided header as a base.
	addNexusHeaderToHTTPHeader(options.Header, request.Header)
	if options.RequestID == "" {
		options.RequestID = uuid.NewString()
	}
	request.Header.Set(headerRequestID, options.RequestID)
	request.Header.Set(headerUserAgent, userAgent)
	addContentHeaderToHTTPHeader(reader.Header, request.Header)

	response, err := c.options.HTTPCaller(request)
	if err != nil {
		return nil, err
	}
	// Do not close response body here to allow successful result to read it.
	if response.StatusCode == http.StatusOK {
		return &ClientStartOperationResult[*LazyValue]{
			Successful: &LazyValue{
				serializer: c.options.Serializer,
				Reader: &Reader{
					Header: httpHeaderToContentHeader(response.Header),
					Reader: response.Body,
				},
			},
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
		return &ClientStartOperationResult[*LazyValue]{
			Pending: &OperationHandle[*LazyValue]{
				Operation: operation,
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

// ExecuteOperationOptions are options for [Client.ExecuteOperation].
type ExecuteOperationOptions struct {
	// Callback URL to provide to the handle for receiving async operation completions. Optional.
	// Even though Client.ExecuteOperation waits for operation completion, some application may want to set this
	// callback as a fallback mechanism.
	CallbackURL string
	// Request ID that may be used by the server handler to dedupe this start request.
	// By default a v4 UUID will be generated by the client.
	RequestID string
	// Header to attach to start and get-result requests. Optional.
	//
	// Header keys with the "content-" prefix are reserved for [Serializer] headers and should not be set in the
	// client API; they are not be avaliable to server [Handler] and [Operation] implementations.
	Header Header
	// Duration to wait for operation completion.
	//
	// ⚠ NOTE: unlike GetOperationResultOptions.Wait, zero and negative values are considered effectively infinite.
	Wait time.Duration
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
func (c *Client) ExecuteOperation(ctx context.Context, operation string, input any, options ExecuteOperationOptions) (*LazyValue, error) {
	so := StartOperationOptions{
		CallbackURL: options.CallbackURL,
		RequestID:   options.RequestID,
		Header:      options.Header,
	}
	result, err := c.StartOperation(ctx, operation, input, so)
	if err != nil {
		return nil, err
	}
	if result.Successful != nil {
		return result.Successful, nil
	}
	handle := result.Pending
	gro := GetOperationResultOptions{
		Header: options.Header,
	}
	if options.Wait <= 0 {
		gro.Wait = time.Duration(math.MaxInt64)
	} else {
		gro.Wait = options.Wait
	}
	return handle.GetResult(ctx, gro)
}

// NewHandle gets a handle to an asynchronous operation by name and ID.
// Does not incur a trip to the server.
// Fails if provided an empty operation or ID.
func (c *Client) NewHandle(operation string, operationID string) (*OperationHandle[*LazyValue], error) {
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
	return &OperationHandle[*LazyValue]{
		client:    c,
		Operation: operation,
		ID:        operationID,
	}, nil
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
	if !isMediaTypeJSON(response.Header.Get("Content-Type")) {
		return nil, newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get("Content-Type")), response, body)
	}
	var info OperationInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func failureFromResponse(response *http.Response, body []byte) (Failure, error) {
	if !isMediaTypeJSON(response.Header.Get("Content-Type")) {
		return Failure{}, newUnexpectedResponseError(fmt.Sprintf("invalid response content type: %q", response.Header.Get("Content-Type")), response, body)
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
