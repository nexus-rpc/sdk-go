package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"
)

var errEmptyOperationName = errors.New("empty operation name")

var errEmptyOperationToken = errors.New("empty operation token")

type (
	// A Client makes Nexus service requests as defined in the [Nexus API].
	//
	// It can start a new operation and get an [OperationHandle] to an existing, asynchronous operation.
	//
	// Use an [OperationHandle] to cancel, get the result of, and get information about asynchronous operations.
	//
	// OperationHandles can be obtained either by starting new operations or by calling [Client.NewHandle] for existing
	// operations.
	//
	// NOTE: Experimental
	//
	// [Nexus API]: https://github.com/nexus-rpc/api
	Client struct {
		options   ClientOptions
		transport Transport
	}

	ClientOptions struct {
		// Service name. Required.
		Service string
	}
)

// NewClient creates a new [Client] from provided [ClientOptions] and [Transport].
// Service is required.
//
// NOTE: Experimental
func NewClient(options ClientOptions, transport Transport) (*Client, error) {
	if options.Service == "" {
		return nil, errors.New("empty Service")
	}
	return &Client{
		options:   options,
		transport: transport,
	}, nil
}

// NewHTTPClient creates a new [HTTPTransport] delegate from provided [HTTPTransportOptions] and uses that and
// provided [ClientOptions] to create a new [Client]. Service is required.
//
// NOTE: Experimental
func NewHTTPClient(copts ClientOptions, topts HTTPTransportOptions) (*Client, error) {
	transport, err := NewHTTPTransport(topts)
	if err != nil {
		return nil, err
	}
	return NewClient(copts, transport)
}

// StartOperation calls the configured Nexus endpoint to start an operation.
//
// This method has the following possible outcomes:
//
//  1. The operation completes successfully. The result of this call will be set as a [LazyValue] in
//     ClientStartOperationResult.Complete and must be consumed to free up the underlying connection.
//
//  2. The operation was started and the handler has indicated that it will complete asynchronously. An
//     [OperationHandle] will be returned as ClientStartOperationResult.Pending, which can be used to perform actions
//     such as getting its result.
//
//  3. The operation was unsuccessful. The returned result will be nil and error will be an
//     [OperationError].
//
//  4. Any other error.
//
// NOTE: Experimental
func (c *Client) StartOperation(
	ctx context.Context,
	operation string,
	input any,
	options StartOperationOptions,
) (*StartOperationResponse[*LazyValue], error) {
	return c.transport.StartOperation(ctx, c.options.Service, operation, input, options)
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
//
// NOTE: Experimental
func (c *Client) ExecuteOperation(
	ctx context.Context,
	operation string,
	input any,
	options ExecuteOperationOptions,
) (*LazyValue, error) {
	so := StartOperationOptions{
		CallbackURL:    options.CallbackURL,
		CallbackHeader: options.CallbackHeader,
		RequestID:      options.RequestID,
		Links:          options.Links,
		Header:         options.Header,
	}
	result, err := c.StartOperation(ctx, operation, input, so)
	if err != nil {
		return nil, err
	}
	if result.Complete != nil {
		return result.Complete.Get()
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
	r, e := handle.GetResult(ctx, gro)
	return r, e
}

// NewHandle gets a handle to an asynchronous operation by name and token.
// Does not incur a trip to the server.
// Fails if provided an empty operation or token.
//
// NOTE: Experimental
func (c *Client) NewHandle(
	operation string,
	token string,
) (*OperationHandle[*LazyValue], error) {
	var es []error
	if operation == "" {
		es = append(es, errEmptyOperationName)
	}
	if token == "" {
		es = append(es, errEmptyOperationToken)
	}
	if len(es) > 0 {
		return nil, errors.Join(es...)
	}
	return &OperationHandle[*LazyValue]{
		transport: c.transport,
		Service:   c.options.Service,
		Operation: operation,
		ID:        token, // Duplicate token as ID for the deprecation period.
		Token:     token,
	}, nil
}

// UnexpectedResponseError indicates a client encountered something unexpected in the server's response.
type UnexpectedResponseError struct {
	// Error message.
	Message string
	// Optional failure that may have been embedded in the response.
	Failure *Failure
	// Additional transport specific details.
	// For HTTP, this would include the HTTP response. The response body will have already been read into memory and
	// does not need to be closed.
	Details any
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
		Message: message,
		Details: response,
		Failure: failure,
	}
}
