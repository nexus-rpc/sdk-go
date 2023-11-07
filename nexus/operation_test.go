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

type asyncNumberValidatorOperationHandler struct {
	UnimplementedOperationHandler[int, int]
}

func (h *asyncNumberValidatorOperationHandler) Name() string {
	return "async-number-validator"
}

func (h *asyncNumberValidatorOperationHandler) Start(ctx context.Context, input int, options StartOperationOptions) (HandlerStartOperationResult[int], error) {
	return &HandlerStartOperationResultAsync{OperationID: "foo"}, nil
}

func (h *asyncNumberValidatorOperationHandler) GetResult(ctx context.Context, id string, options GetOperationResultOptions) (int, error) {
	return 3, nil
}

func (h *asyncNumberValidatorOperationHandler) Cancel(ctx context.Context, id string, options CancelOperationOptions) error {
	if options.Header.Get("fail") != "" {
		return fmt.Errorf("intentionally failed")
	}
	return nil
}

func (h *asyncNumberValidatorOperationHandler) GetInfo(ctx context.Context, id string, options GetOperationInfoOptions) (*OperationInfo, error) {
	if options.Header.Get("fail") != "" {
		return nil, fmt.Errorf("intentionally failed")
	}
	return &OperationInfo{ID: id, State: OperationStateRunning}, nil
}

var asyncNumberValidatorOperation = &asyncNumberValidatorOperationHandler{}

func TestOperationDirectory(t *testing.T) {
	options := OperationDirectoryHandlerOptions{
		Operations: []UntypedOperationHandler{
			numberValidatorOperation,
			numberValidatorOperation,
		},
	}

	_, err := NewOperationDirectoryHandler(options)
	require.ErrorContains(t, err, "duplicate operations: "+numberValidatorOperation.Name())
	options.Operations = nil
	_, err = NewOperationDirectoryHandler(options)
	require.ErrorContains(t, err, "must register at least one operation")
}

func TestExecuteOperation(t *testing.T) {
	options := OperationDirectoryHandlerOptions{
		Operations: []UntypedOperationHandler{
			numberValidatorOperation,
			bytesIOOperation,
			noValueOperation,
		},
	}

	handler, err := NewOperationDirectoryHandler(options)
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := ExecuteOperation(ctx, client, numberValidatorOperation, 3, ExecuteOperationOptions{})
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
	asyncNumberValidatorOperation := &asyncNumberValidatorOperationHandler{}
	options := OperationDirectoryHandlerOptions{
		Operations: []UntypedOperationHandler{
			numberValidatorOperation,
			asyncNumberValidatorOperation,
		},
	}

	handler, err := NewOperationDirectoryHandler(options)
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := StartOperation(ctx, client, numberValidatorOperation, 3, StartOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, result.Successful)

	result, err = StartOperation(ctx, client, asyncNumberValidatorOperation, 3, StartOperationOptions{})
	require.NoError(t, err)
	value, err := result.Pending.GetResult(ctx, GetOperationResultOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, value)
}

func TestCancelOperation(t *testing.T) {
	options := OperationDirectoryHandlerOptions{
		Operations: []UntypedOperationHandler{
			asyncNumberValidatorOperation,
		},
	}

	handler, err := NewOperationDirectoryHandler(options)
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := StartOperation(ctx, client, asyncNumberValidatorOperation, 3, StartOperationOptions{})
	require.NoError(t, err)
	require.NoError(t, result.Pending.Cancel(ctx, CancelOperationOptions{}))
	var unexpectedError *UnexpectedResponseError
	require.ErrorAs(t, result.Pending.Cancel(ctx, CancelOperationOptions{Header: Header{"fail": "1"}}), &unexpectedError)
}

func TestGetOperationInfo(t *testing.T) {
	options := OperationDirectoryHandlerOptions{
		Operations: []UntypedOperationHandler{
			asyncNumberValidatorOperation,
		},
	}

	handler, err := NewOperationDirectoryHandler(options)
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := StartOperation(ctx, client, asyncNumberValidatorOperation, 3, StartOperationOptions{})
	require.NoError(t, err)
	info, err := result.Pending.GetInfo(ctx, GetOperationInfoOptions{})
	require.NoError(t, err)
	require.Equal(t, &OperationInfo{ID: "foo", State: OperationStateRunning}, info)
	_, err = result.Pending.GetInfo(ctx, GetOperationInfoOptions{Header: Header{"fail": "1"}})
	var unexpectedError *UnexpectedResponseError
	require.ErrorAs(t, err, &unexpectedError)
}
