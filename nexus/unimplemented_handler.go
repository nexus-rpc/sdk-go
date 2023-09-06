package nexus

import (
	"context"
	"net/http"
)

// UnimplementedHandler must be embedded into any [Handler] implementation for future compatibility.
// It implements all methods on the [Handler] interface, panicking at runtime if they are not implemented by the
// embedding type.
type UnimplementedHandler struct{}

func (h *UnimplementedHandler) mustEmbedUnimplementedHandler() {}

// StartOperation implements the Handler interface.
func (h *UnimplementedHandler) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	return nil, &HandlerError{http.StatusNotImplemented, &Failure{Message: "not implemented"}}
}

// GetOperationResult implements the Handler interface.
func (h *UnimplementedHandler) GetOperationResult(ctx context.Context, request *GetOperationResultRequest) (*OperationResponseSync, error) {
	return nil, &HandlerError{http.StatusNotImplemented, &Failure{Message: "not implemented"}}
}

// GetOperationInfo implements the Handler interface.
func (h *UnimplementedHandler) GetOperationInfo(ctx context.Context, request *GetOperationInfoRequest) (*OperationInfo, error) {
	return nil, &HandlerError{http.StatusNotImplemented, &Failure{Message: "not implemented"}}
}

// CancelOperation implements the Handler interface.
func (h *UnimplementedHandler) CancelOperation(ctx context.Context, request *CancelOperationRequest) error {
	return &HandlerError{http.StatusNotImplemented, &Failure{Message: "not implemented"}}
}
