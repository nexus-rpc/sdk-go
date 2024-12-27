package nexus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type executeOperationHandler struct {
	UnimplementedOperation[string, string]
}

func (*executeOperationHandler) Name() string {
	return "execute"
}

// Start implements Operation.
func (*executeOperationHandler) Start(ctx context.Context, input string, options StartOperationOptions) (HandlerStartOperationResult[string], error) {
	return &HandlerStartOperationResultAsync{
		OperationID: input,
		StartResult: "some-run-id",
	}, nil
}

// GetResult implements Operation.
func (*executeOperationHandler) GetResult(ctx context.Context, id string, _ GetOperationResultOptions) (string, error) {
	if id != "some-id" {
		return "", HandlerErrorf(HandlerErrorTypeBadRequest, "invalid ID")
	}
	return "complete", nil
}

func TestStartResult(t *testing.T) {
	svc := NewService(testService)
	op := &executeOperationHandler{}
	require.NoError(t, svc.Register(op))
	reg := NewServiceRegistry()
	require.NoError(t, reg.Register(svc))
	handler, err := reg.NewHandler()
	require.NoError(t, err)
	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := StartOperation(ctx, client, op, "some-id", StartOperationOptions{})
	require.NoError(t, err)
	var runID string
	require.NoError(t, result.StartResult.Consume(&runID))
	outcome, err := result.Pending.GetResult(ctx, GetOperationResultOptions{})
	require.NoError(t, err)
	require.Equal(t, "complete", outcome)
}
