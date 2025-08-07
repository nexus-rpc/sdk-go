package nexus

import (
	"context"
	"errors"
	"math"
	"time"
)

var errEmptyOperationName = errors.New("empty operation name")

var errEmptyOperationToken = errors.New("empty operation token")

// A ServiceClient makes Nexus service requests to start new operations or get an [OperationHandle] to an
// existing, asynchronous operation.
//
// Use an [OperationHandle] to cancel, get the result of, and get information about asynchronous operations.
//
// OperationHandles can be obtained either by starting new operations or by calling [ServiceClient.NewOperationHandle] for existing
// operations.
//
// NOTE: Experimental
type ServiceClient struct {
	options ServiceClientOptions
}

// ServiceClientOptions are the options for creating a new [ServiceClient].
type ServiceClientOptions struct {
	// Service name. Required.
	Service string
	// Transport delegate for making network calls. Required.
	Transport Transport
}

// NewServiceClient creates a new [ServiceClient] from provided [ServiceClientOptions].
// Service and Transport are required.
//
// NOTE: Experimental
func NewServiceClient(options ServiceClientOptions) (*ServiceClient, error) {
	if options.Service == "" {
		return nil, errors.New("empty Service")
	}
	if options.Transport == nil {
		return nil, errors.New("nil Transport")
	}
	return &ServiceClient{
		options: options,
	}, nil
}

// StartOperation calls the configured Nexus endpoint to start an operation.
//
// This method has the following possible outcomes:
//
//  1. The operation completes successfully. The result of this call will be set as a [LazyValue] in
//     ClientStartOperationResponse.Complete and must be consumed to free up the underlying connection.
//
//  2. The operation completes unsuccessfully. The response will contain an [OperationError] in
//     ClientStartOperationResponse.Complete
//
//  3. The operation was started and the handler has indicated that it will complete asynchronously. An
//     [OperationHandle] will be returned as ClientStartOperationResponse.Pending, which can be used to perform actions
//     such as getting its result.
//
//  4. There was an error making the call. The returned response will be nil and the error will be non-nil.
//     Most often it will be a [HandlerError].
//
// NOTE: Experimental
func (c *ServiceClient) StartOperation(
	ctx context.Context,
	operation string,
	input any,
	options StartOperationOptions,
) (*ClientStartOperationResponse[*LazyValue], error) {
	to := TransportStartOperationOptions{
		ClientOptions: options,
		Service:       c.options.Service,
		Operation:     operation,
	}
	resp, err := c.options.Transport.StartOperation(ctx, input, to)
	if err != nil {
		return nil, err
	}
	switch r := resp.Variant.(type) {
	case *TransportStartOperationResponseSync[*LazyValue]:
		return &ClientStartOperationResponse[*LazyValue]{
			Links: resp.Links,
			Variant: &ClientStartOperationResponseSync[*LazyValue]{
				Success: r.Success,
			},
		}, nil
	case *TransportStartOperationResponseAsync[*LazyValue]:
		return &ClientStartOperationResponse[*LazyValue]{
			Links: resp.Links,
			Variant: &ClientStartOperationResponseAsync[*LazyValue]{
				Running: r.Running,
			},
		}, nil
	}

	return nil, HandlerErrorf(HandlerErrorTypeInternal, "transport returned unexpected response type: %T", resp.Variant)
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
func (c *ServiceClient) ExecuteOperation(
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
	response, err := c.StartOperation(ctx, operation, input, so)
	if err != nil {
		return nil, err
	}
	if response.Sync() != nil {
		return response.Sync().Get()
	}

	if response.Async() == nil {
		return nil, HandlerErrorf(HandlerErrorTypeInternal, "transport returned unexpected response type: %T", response.Variant)
	}
	handle := response.Async()
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

// NewOperationHandle gets a handle to an asynchronous operation by name and token.
// Does not incur a trip to the server.
// Fails if provided an empty operation or token.
//
// NOTE: Experimental
func (c *ServiceClient) NewOperationHandle(
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
		transport: c.options.Transport,
		Service:   c.options.Service,
		Operation: operation,
		ID:        token, // Duplicate token as ID for the deprecation period.
		Token:     token,
	}, nil
}

// isClientStartOperationResponseVariant must be implemented by each possible [ServiceClient.StartOperation] response
// variant. This enforces one and only one variant will be set and allows for type checking to get the value of it.
//
// NOTE: Experimental
type isClientStartOperationResponseVariant[T any] interface {
	isClientStartOperationResponseVariant()
}

// ClientStartOperationResponse is the response to ServiceClient.StartOperation calls.
//
// NOTE: Experimental
type ClientStartOperationResponse[T any] struct {
	Variant isClientStartOperationResponseVariant[T]
	// Links contain information about the operations done by the handler.
	Links []Link
}

// Sync is non-nil when start completes synchronously.
//
// If T is a [LazyValue], ensure that your consume it or read the underlying content in its entirety and close
// it to free up the underlying connection.
//
// NOTE: Experimental
func (r *ClientStartOperationResponse[T]) Sync() *OperationResult[T] {
	if s, ok := r.Variant.(*ClientStartOperationResponseSync[T]); ok {
		return s.Success
	}
	return nil
}

// Async is non-nil when the handler indicates that it started an asynchronous operation.
// The attached handle can be used to perform actions such as cancel the operation or get its result.
//
// NOTE: Experimental
func (r *ClientStartOperationResponse[T]) Async() *OperationHandle[T] {
	if a, ok := r.Variant.(*ClientStartOperationResponseAsync[T]); ok {
		return a.Running
	}
	return nil
}

// ClientStartOperationResponseSync is the response variant for [ServiceClient.StartOperation] that indicates the
// operation completed synchronously.
//
// If T is a [LazyValue], ensure that your consume it or read the underlying content in its entirety and close
// it to free up the underlying connection.
//
// NOTE: Experimental
type ClientStartOperationResponseSync[T any] struct {
	Success *OperationResult[T]
}

// ClientStartOperationResponseAsync is the response variant for [ServiceClient.StartOperation] that indicates the
// operation will complete asynchronously. The attached handle can be used to perform actions such as cancel the
// operation or get its result.
//
// NOTE: Experimental
type ClientStartOperationResponseAsync[T any] struct {
	Running *OperationHandle[T]
}

func (*ClientStartOperationResponseSync[T]) isClientStartOperationResponseVariant()  {} //nolint:unused
func (*ClientStartOperationResponseAsync[T]) isClientStartOperationResponseVariant() {} //nolint:unused
