package nexus

import (
	"context"
)

// UnimplementedHandler must be embedded into any [Handler] implementation for future compatibility.
// It implements all methods on the [Handler] interface, panicking at runtime if they are not implemented by the
// embedding type.
type UnimplementedHandler struct{}

func (h *UnimplementedHandler) mustEmbedUnimplementedHandler() {}

// StartOperation implements the Handler interface.
func (h *UnimplementedHandler) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	panic("unimplemented")
}

// GetOperationResult implements the Handler interface.
func (h *UnimplementedHandler) GetOperationResult(ctx context.Context, request *GetOperationResultRequest) (OperationResponse, error) {
	panic("unimplemented")
}

// GetOperationInfo implements the Handler interface.
func (h *UnimplementedHandler) GetOperationInfo(ctx context.Context, request *GetOperationInfoRequest) (*OperationInfo, error) {
	panic("unimplemented")
}

// CancelOperation implements the Handler interface.
func (h *UnimplementedHandler) CancelOperation(ctx context.Context, request *CancelOperationRequest) error {
	panic("unimplemented")
}
