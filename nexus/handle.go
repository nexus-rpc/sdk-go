package nexus

import (
	"context"
)

type (
	OperationClient interface {
		GetOperationInfo(ctx context.Context, operation string, token string, options GetOperationInfoOptions) (*OperationInfo, error)
		GetOperationResult(ctx context.Context, operation string, token string, options GetOperationResultOptions) (*FullResult[*LazyValue], error)
		CancelOperation(ctx context.Context, operation string, token string, options CancelOperationOptions) error
	}

	// An OperationHandle is used to cancel operations and get their result and status.
	OperationHandle[T any] struct {
		// Name of the Service to which this operation belongs.
		Service string
		// Name of the Operation this handle represents.
		Operation string
		// Handler generated ID for this handle's operation.
		//
		// Deprecated: Use Token instead.
		ID string
		// Handler generated token for this handle's operation.
		Token string

		client OperationClient
	}

	FullResult[T any] struct {
		Links  []Link
		Result T
	}
)

// GetInfo gets operation information, issuing a network request to the service handler.
//
// NOTE: Experimental
func (h *OperationHandle[T]) GetInfo(ctx context.Context, options GetOperationInfoOptions) (*OperationInfo, error) {
	t := h.Token
	if h.Token != h.ID {
		t = h.ID
	}
	return h.client.GetOperationInfo(ctx, h.Operation, t, options)
}

// GetResult gets the result of an operation, issuing a network request to the service handler.
//
// By default, GetResult returns (nil, [ErrOperationStillRunning]) immediately after issuing a call if the operation has
// not yet completed.
//
// Callers may set GetOperationResultOptions.Wait to a value greater than 0 to alter this behavior, causing the client
// to long poll for the result issuing one or more requests until the provided wait period exceeds, in which case (nil,
// [ErrOperationStillRunning]) is returned.
//
// The wait time is capped to the deadline of the provided context. Make sure to handle both context deadline errors and
// [ErrOperationStillRunning].
//
// Note that the wait period is enforced by the server and may not be respected if the server is misbehaving. Set the
// context deadline to the max allowed wait period to ensure this call returns in a timely fashion.
//
// ⚠️ If a [LazyValue] is returned (as indicated by T), it must be consumed to free up the underlying connection.
//
// NOTE: Experimental
func (h *OperationHandle[T]) GetResult(ctx context.Context, options GetOperationResultOptions) (T, error) {
	var result T
	full, err := h.GetFullResult(ctx, options)
	if err != nil {
		return result, err
	}
	return full.Result, nil
}

func (h *OperationHandle[T]) GetFullResult(ctx context.Context, options GetOperationResultOptions) (*FullResult[T], error) {
	var result T
	t := h.Token
	if h.Token != h.ID {
		t = h.ID
	}
	s, err := h.client.GetOperationResult(ctx, h.Operation, t, options)
	if err != nil {
		return nil, err
	}
	if _, ok := any(result).(*LazyValue); ok {
		return &FullResult[T]{
			Links:  s.Links,
			Result: any(s.Result).(T),
		}, nil
	} else {
		return &FullResult[T]{
			Links:  s.Links,
			Result: result,
		}, s.Result.Consume(&result)
	}
}

// Cancel requests to cancel an asynchronous operation.
//
// Cancelation is asynchronous and may be not be respected by the operation's implementation.
func (h *OperationHandle[T]) Cancel(ctx context.Context, options CancelOperationOptions) error {
	t := h.Token
	if h.Token != h.ID {
		t = h.ID
	}
	return h.client.CancelOperation(ctx, h.Operation, t, options)
}
