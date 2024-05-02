package nexus

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var bytesIOOperation = NewSyncOperation("bytes-io", func(ctx context.Context, input []byte, options StartOperationOptions) ([]byte, error) {
	return append(input, []byte(", world")...), nil
})

var noValueOperation = NewSyncOperation("no-value", func(ctx context.Context, input NoValue, options StartOperationOptions) (NoValue, error) {
	return nil, nil
})

var numberValidatorOperation = NewSyncOperation("number-validator", func(ctx context.Context, input int, options StartOperationOptions) (int, error) {
	if input == 0 {
		return 0, &UnsuccessfulOperationError{State: OperationStateFailed, Failure: Failure{Message: "cannot process 0"}}
	}
	return input, nil
})

type asyncNumberValidatorOperation struct {
	UnimplementedOperation[int, int]
}

func (h *asyncNumberValidatorOperation) Name() string {
	return "async-number-validator"
}

func (h *asyncNumberValidatorOperation) Start(ctx context.Context, input int, options StartOperationOptions) (HandlerStartOperationResult[int], error) {
	return &HandlerStartOperationResultAsync{OperationID: "foo"}, nil
}

func (h *asyncNumberValidatorOperation) GetResult(ctx context.Context, id string, options GetOperationResultOptions) (int, error) {
	return 3, nil
}

func (h *asyncNumberValidatorOperation) Cancel(ctx context.Context, id string, options CancelOperationOptions) error {
	if options.Header.Get("fail") != "" {
		return fmt.Errorf("intentionally failed")
	}
	return nil
}

func (h *asyncNumberValidatorOperation) GetInfo(ctx context.Context, id string, options GetOperationInfoOptions) (*OperationInfo, error) {
	if options.Header.Get("fail") != "" {
		return nil, fmt.Errorf("intentionally failed")
	}
	return &OperationInfo{ID: id, State: OperationStateRunning}, nil
}

var asyncNumberValidatorOperationInstance = &asyncNumberValidatorOperation{}

func TestRegistrationErrors(t *testing.T) {
	reg := NewServiceRegistry()
	svc := NewService(testService)
	err := svc.Register(NewSyncOperation("", func(ctx context.Context, i int, soo StartOperationOptions) (int, error) { return 5, nil }))
	require.ErrorContains(t, err, "tried to register an operation with no name")

	err = svc.Register(numberValidatorOperation, numberValidatorOperation)
	require.ErrorContains(t, err, "duplicate operations: "+numberValidatorOperation.Name())

	_, err = reg.NewHandler()
	require.ErrorContains(t, err, "must register at least one service")

	require.ErrorContains(t, reg.Register(NewService("")), "tried to register a service with no name")
	// Reset operations to trigger an error.
	svc.operations = nil
	require.NoError(t, reg.Register(svc))

	_, err = reg.NewHandler()
	require.ErrorContains(t, err, fmt.Sprintf("service %q has no operations registered", testService))
}

func TestExecuteOperation(t *testing.T) {
	registry := NewServiceRegistry()
	svc := NewService(testService)
	require.NoError(t, svc.Register(
		numberValidatorOperation,
		bytesIOOperation,
		noValueOperation,
	))
	require.NoError(t, registry.Register(svc))
	handler, err := registry.NewHandler()
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := ExecuteOperation(ctx, client, numberValidatorOperation, 3, ExecuteOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, result)

	ref := NewOperationReference[int, int](numberValidatorOperation.Name())
	result, err = ExecuteOperation(ctx, client, ref, 3, ExecuteOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, result)

	_, err = ExecuteOperation(ctx, client, numberValidatorOperation, 0, ExecuteOperationOptions{})
	var unsuccessfulError *UnsuccessfulOperationError
	require.ErrorAs(t, err, &unsuccessfulError)

	bResult, err := ExecuteOperation(ctx, client, bytesIOOperation, []byte("hello"), ExecuteOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, []byte("hello, world"), bResult)

	nResult, err := ExecuteOperation(ctx, client, noValueOperation, nil, ExecuteOperationOptions{})
	require.NoError(t, err)
	require.Nil(t, nResult)
}

func TestStartOperation(t *testing.T) {
	registry := NewServiceRegistry()
	svc := NewService(testService)
	require.NoError(t, svc.Register(
		numberValidatorOperation,
		asyncNumberValidatorOperationInstance,
	))
	require.NoError(t, registry.Register(svc))

	handler, err := registry.NewHandler()
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := StartOperation(ctx, client, numberValidatorOperation, 3, StartOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, result.Successful)

	result, err = StartOperation(ctx, client, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{})
	require.NoError(t, err)
	value, err := result.Pending.GetResult(ctx, GetOperationResultOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, value)
	handle, err := NewHandle(client, asyncNumberValidatorOperationInstance, result.Pending.ID)
	require.NoError(t, err)
	value, err = handle.GetResult(ctx, GetOperationResultOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, value)
}

func TestCancelOperation(t *testing.T) {
	registry := NewServiceRegistry()
	svc := NewService(testService)
	require.NoError(t, svc.Register(
		asyncNumberValidatorOperationInstance,
	))
	require.NoError(t, registry.Register(svc))

	handler, err := registry.NewHandler()
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := StartOperation(ctx, client, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{})
	require.NoError(t, err)
	require.NoError(t, result.Pending.Cancel(ctx, CancelOperationOptions{}))
	var unexpectedError *UnexpectedResponseError
	require.ErrorAs(t, result.Pending.Cancel(ctx, CancelOperationOptions{Header: Header{"fail": "1"}}), &unexpectedError)
}

func TestGetOperationInfo(t *testing.T) {
	registry := NewServiceRegistry()
	svc := NewService(testService)
	require.NoError(t, svc.Register(
		asyncNumberValidatorOperationInstance,
	))
	require.NoError(t, registry.Register(svc))

	handler, err := registry.NewHandler()
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := StartOperation(ctx, client, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{})
	require.NoError(t, err)
	info, err := result.Pending.GetInfo(ctx, GetOperationInfoOptions{})
	require.NoError(t, err)
	require.Equal(t, &OperationInfo{ID: "foo", State: OperationStateRunning}, info)
	_, err = result.Pending.GetInfo(ctx, GetOperationInfoOptions{Header: Header{"fail": "1"}})
	var unexpectedError *UnexpectedResponseError
	require.ErrorAs(t, err, &unexpectedError)
}
