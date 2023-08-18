package test

import (
	"context"

	"github.com/nexus-rpc/sdk-go/nexusapi"
	"github.com/nexus-rpc/sdk-go/nexusserver"
)

type unimplementedHandler struct{}

func (h *unimplementedHandler) StartOperation(ctx context.Context, writer nexusserver.ResultWriter, request *nexusserver.StartOperationRequest) error {
	panic("unimplemented")
}

func (h *unimplementedHandler) GetOperationResult(ctx context.Context, writer nexusserver.ResultWriter, request *nexusserver.GetOperationResultRequest) error {
	panic("unimplemented")
}

func (h *unimplementedHandler) GetOperationInfo(ctx context.Context, request *nexusserver.GetOperationInfoRequest) (nexusapi.OperationInfo, error) {
	panic("unimplemented")
}

func (h *unimplementedHandler) CancelOperation(ctx context.Context, request *nexusserver.CancelOperationRequest) error {
	panic("unimplemented")
}
