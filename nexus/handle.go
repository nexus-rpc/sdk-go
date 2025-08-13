package nexus

import (
	"context"
)

// An OperationHandle is used to cancel operations and get their result and status.
type OperationHandle[T any] struct {
	// Service to which this operation belongs.
	Service string
	// Name of the Operation this handle represents.
	Operation string
	// Handler generated ID for this handle's operation.
	//
	// Deprecated: Use Token instead.
	ID string
	// Handler generated token for this handle's operation.
	Token string

	// transport is used to make external network calls.
	transport Transport
}

// FetchInfo gets operation information, issuing a network request to the service handler.
//
// NOTE: Experimental
func (h *OperationHandle[T]) FetchInfo(ctx context.Context, options FetchOperationInfoOptions) (*OperationInfo, error) {
	to := TransportFetchOperationInfoOptions{
		ClientOptions: options,
		Service:       h.Service,
		Operation:     h.Operation,
		Token:         h.Token,
	}
	resp, err := h.transport.FetchOperationInfo(ctx, to)
	if err != nil {
		return nil, err
	}
	return resp.Info, nil
}

// FetchResult gets the result of an operation, issuing a network request to the service handler.
//
// This is a convenience method on top of FetchResultWithDetails for callers who do not wish to inspect metadata.
//
// The returned error may be an [OperationError] returned by the handler, indicating the operation completed
// unsuccessfully, a [HandlerError] indicating a failure to communicate with the handler, or any other error.
//
// NOTE: Experimental
func (h *OperationHandle[T]) FetchResult(ctx context.Context, options FetchOperationResultOptions) (T, error) {
	var result T
	res, err := h.FetchResultWithDetails(ctx, options)
	if err != nil {
		return result, err
	}
	return res.Get()
}

// FetchResultWithDetails gets the result of an operation and associated metadata, issuing a network request to the service
// handler.
//
// By default, FetchOperationResult returns (nil, [ErrOperationStillRunning]) immediately after issuing a call if the
// operation has not yet completed.
//
// Callers may set FetchOperationResultOptions.Wait to a value greater than 0 to alter this behavior, causing the client
// to long poll for the result issuing one or more requests until the provided wait period exceeds, in which case (nil,
// [ErrOperationStillRunning]) is returned.
//
// The wait time is capped to the deadline of the provided context. Make sure to handle both context deadline errors and
// [ErrOperationStillRunning].
//
// Note that the wait period is enforced by the server and may not be respected if the server is misbehaving. Set the
// context deadline to the max allowed wait period to ensure this call returns in a timely fashion.
//
// Errors returned by the method itself indicate a failure to communicate with the operation handler and are typically
// represented by a [HandlerError].
//
// # The final value or error returned by the operation can be retrieved with OperationHandleResultWithDetails.Get
//
// ⚠️ If a [LazyValue] is returned (as indicated by T), it must be consumed to free up the underlying connection.
//
// NOTE: Experimental
func (h *OperationHandle[T]) FetchResultWithDetails(ctx context.Context, options FetchOperationResultOptions) (*OperationHandleResultWithDetails[T], error) {
	to := TransportFetchOperationResultOptions{
		ClientOptions: options,
		Service:       h.Service,
		Operation:     h.Operation,
		Token:         h.Token,
	}
	resp, err := h.transport.FetchOperationResult(ctx, to)
	if err != nil {
		return nil, err
	}
	lv, err := resp.Result()
	if err != nil {
		return &OperationHandleResultWithDetails[T]{
			result: &OperationResult[T]{
				err: err,
			},
			Links: resp.Links,
		}, nil
	}

	var result T
	if _, ok := any(result).(*LazyValue); ok {
		return &OperationHandleResultWithDetails[T]{
			result: &OperationResult[T]{
				result: any(lv).(T),
			},
			Links: resp.Links,
		}, nil
	}

	return &OperationHandleResultWithDetails[T]{
		result: &OperationResult[T]{
			result: result,
			err:    lv.Consume(&result),
		},
		Links: resp.Links,
	}, nil
}

// Cancel requests to cancel an asynchronous operation.
//
// Cancelation is asynchronous and may be not be respected by the operation's implementation.
func (h *OperationHandle[T]) Cancel(ctx context.Context, options CancelOperationOptions) error {
	to := TransportCancelOperationOptions{
		ClientOptions: options,
		Service:       h.Service,
		Operation:     h.Operation,
		Token:         h.Token,
	}
	_, err := h.transport.CancelOperation(ctx, to)
	return err
}

// OperationHandleResultWithDetails is a wrapper for the result of an operation with any associated metadata.
//
// NOTE: Experimental
type OperationHandleResultWithDetails[T any] struct {
	result *OperationResult[T]
	Links  []Link
}

// Get returns the final result or error returned by the operation.
//
// NOTE: Experimental
func (r *OperationHandleResultWithDetails[T]) Get() (T, error) {
	return r.result.Get()
}
