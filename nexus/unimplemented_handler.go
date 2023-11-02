package nexus

import (
	"context"
)

// UnimplementedServiceHandler must be embedded into any [ServiceHandler] implementation for future compatibility.
// It implements all methods on the [ServiceHandler] interface, panicking at runtime if they are not implemented by the
// embedding type.
type UnimplementedServiceHandler struct{}

func (h UnimplementedServiceHandler) mustEmbedUnimplementedHandler() {}

// StartOperation implements the ServiceHandler interface.
func (h UnimplementedServiceHandler) StartOperation(ctx context.Context, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// GetOperationResult implements the ServiceHandler interface.
func (h UnimplementedServiceHandler) GetOperationResult(ctx context.Context, operation, operationID string, options GetOperationResultOptions) (any, error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// GetOperationInfo implements the ServiceHandler interface.
func (h UnimplementedServiceHandler) GetOperationInfo(ctx context.Context, operation, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// CancelOperation implements the ServiceHandler interface.
func (h UnimplementedServiceHandler) CancelOperation(ctx context.Context, operation, operationID string, options CancelOperationOptions) error {
	return &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// UnimplementedOperationHandler must be embedded into any [OperationHandler] implementation for future compatibility.
// It implements all methods on the [OperationHandler] interface except for `Name`, panicking at runtime if they are not
// implemented by the embedding type.
type UnimplementedOperationHandler[I, O any] struct{}

func (*UnimplementedOperationHandler[I, O]) inferType(I, O) {}

func (*UnimplementedOperationHandler[I, O]) mustEmbedUnimplementedOperationHandler() {}

// Cancel implements OperationHandler.
func (*UnimplementedOperationHandler[I, O]) Cancel(context.Context, string, CancelOperationOptions) error {
	return HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// GetInfo implements OperationHandler.
func (*UnimplementedOperationHandler[I, O]) GetInfo(context.Context, string, GetOperationInfoOptions) (*OperationInfo, error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// GetResult implements OperationHandler.
func (*UnimplementedOperationHandler[I, O]) GetResult(context.Context, string, GetOperationResultOptions) (O, error) {
	var empty O
	return empty, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// Start implements OperationHandler.
func (h *UnimplementedOperationHandler[I, O]) Start(ctx context.Context, input I, options StartOperationOptions) (HandlerStartOperationResult[O], error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}
