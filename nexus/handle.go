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

	transport Transport
}

// GetInfo gets operation information, issuing a network request to the service handler.
//
// NOTE: Experimental
func (h *OperationHandle[T]) GetInfo(ctx context.Context, options GetOperationInfoOptions) (*OperationInfo, error) {
	resp, err := h.GetInfoResponse(ctx, options)
	if err != nil {
		return nil, err
	}
	return resp.Info, nil
}

func (h *OperationHandle[T]) GetInfoResponse(ctx context.Context, options GetOperationInfoOptions) (*GetOperationInfoResponse, error) {
	return h.transport.GetOperationInfo(ctx, h.Service, h.Operation, h.Token, options)
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
	resp, err := h.GetResultResponse(ctx, options)
	if err != nil {
		return result, err
	}
	r, e := resp.GetResult()
	return r, e
}

func (h *OperationHandle[T]) GetResultResponse(ctx context.Context, options GetOperationResultOptions) (*GetOperationResultResponse[T], error) {
	resp, err := h.transport.GetOperationResult(ctx, h.Service, h.Operation, h.Token, options)
	if err != nil {
		return nil, err
	}
	lv, err := resp.GetResult()
	if err != nil {
		return &GetOperationResultResponse[T]{
			result: &OperationResult[T]{
				err: err,
			},
			Links: resp.Links,
		}, nil
	}

	var result T
	if _, ok := any(result).(*LazyValue); ok {
		return &GetOperationResultResponse[T]{
			result: &OperationResult[T]{
				result: any(lv).(T),
			},
			Links: resp.Links,
		}, nil
	}

	return &GetOperationResultResponse[T]{
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
	_, err := h.CancelResponse(ctx, options)
	return err
}

func (h *OperationHandle[T]) CancelResponse(ctx context.Context, options CancelOperationOptions) (*CancelOperationResponse, error) {
	return h.transport.CancelOperation(ctx, h.Service, h.Operation, h.Token, options)
}
