package nexus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type startOperationHandler struct {
	UnimplementedOperation[string, string]
}

func (*startOperationHandler) Name() string {
	return "start"
}

// Start implements Operation.
func (h *startOperationHandler) Start(ctx context.Context, input string, options StartOperationOptions) (HandlerStartOperationResult[string], error) {
	return &HandlerStartOperationResultSync[string]{
		Value: "some-run-id",
		NextStage: &OperationInfo{
			Name: "wait-complete",
			ID:   input,
			// This should just be the default.
			State: OperationStateRunning,
		},
	}, nil
}

type waitCompleteOperationHandler struct {
	UnimplementedOperation[NoValue, string]
}

func (*waitCompleteOperationHandler) Name() string {
	return "wait-complete"
}

// GetResult implements Operation.
func (*waitCompleteOperationHandler) GetResult(ctx context.Context, id string, _ GetOperationResultOptions) (string, error) {
	if id != "some-id" {
		return "", HandlerErrorf(HandlerErrorTypeBadRequest, "invalid ID")
	}
	return "complete", nil
}

func TestNextStage(t *testing.T) {
	svc := NewService(testService)
	startOp, waitCompleteOp := &startOperationHandler{}, &waitCompleteOperationHandler{}
	require.NoError(t, svc.Register(startOp, waitCompleteOp))
	reg := NewServiceRegistry()
	require.NoError(t, reg.Register(svc))
	handler, err := reg.NewHandler()
	require.NoError(t, err)
	ctx, client, teardown := setup(t, handler)
	defer teardown()

	startRes, err := StartOperation(ctx, client, startOp, "some-id", StartOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, "some-run-id", startRes.Successful)
	handle := NextStage(startRes, waitCompleteOp)
	require.NotNil(t, handle)
	waitRes, err := handle.GetResult(ctx, GetOperationResultOptions{})
	require.NoError(t, err)
	require.NotNil(t, handle)
	require.Equal(t, "complete", waitRes)
}
