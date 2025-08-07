package nexus

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
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
		return 0, NewOperationFailedError("cannot process 0")
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
	return &HandlerStartOperationResultAsync{OperationID: fmt.Sprintf("%d", input)}, nil
}

func (h *asyncNumberValidatorOperation) GetResult(ctx context.Context, token string, options GetOperationResultOptions) (int, error) {
	if token == "0" {
		return 0, NewOperationFailedError("cannot process 0")
	}
	return strconv.Atoi(token)
}

func (h *asyncNumberValidatorOperation) Cancel(ctx context.Context, token string, options CancelOperationOptions) error {
	if options.Header.Get("fail") != "" {
		return fmt.Errorf("intentionally failed")
	}
	return nil
}

func (h *asyncNumberValidatorOperation) GetInfo(ctx context.Context, token string, options GetOperationInfoOptions) (*OperationInfo, error) {
	if options.Header.Get("fail") != "" {
		return nil, fmt.Errorf("intentionally failed")
	}
	return &OperationInfo{Token: token, State: OperationStateRunning}, nil
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
	var unsuccessfulError *OperationError
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
	val, err := result.Sync().Get()
	require.NoError(t, err)
	require.Equal(t, 3, val)

	result, err = StartOperation(ctx, client, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{})
	require.NoError(t, err)
	value, err := result.Async().GetResult(ctx, GetOperationResultOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, value)
	handle, err := NewOperationHandle(client, asyncNumberValidatorOperationInstance, result.Async().Token)
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
	require.NoError(t, result.Async().Cancel(ctx, CancelOperationOptions{}))
	var handlerError *HandlerError
	require.ErrorAs(t, result.Async().Cancel(ctx, CancelOperationOptions{Header: Header{"fail": "1"}}), &handlerError)
	require.Equal(t, HandlerErrorTypeInternal, handlerError.Type)
	require.Equal(t, "internal server error", handlerError.Cause.Error())
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
	info, err := result.Async().GetInfo(ctx, GetOperationInfoOptions{})
	require.NoError(t, err)
	require.Equal(t, &OperationInfo{Token: "3", ID: "3", State: OperationStateRunning}, info)
	_, err = result.Async().GetInfo(ctx, GetOperationInfoOptions{Header: Header{"fail": "1"}})
	var handlerError *HandlerError
	require.ErrorAs(t, err, &handlerError)
	require.Equal(t, HandlerErrorTypeInternal, handlerError.Type)
	require.Equal(t, "internal server error", handlerError.Cause.Error())
}

type authRejectionHandler struct {
	UnimplementedOperation[NoValue, NoValue]
}

func (h *authRejectionHandler) Name() string {
	return "async-number-validator"
}

func (h *authRejectionHandler) Start(ctx context.Context, input NoValue, options StartOperationOptions) (HandlerStartOperationResult[NoValue], error) {
	return nil, HandlerErrorf(HandlerErrorTypeUnauthorized, "unauthorized in test")
}

func (h *authRejectionHandler) GetResult(ctx context.Context, token string, options GetOperationResultOptions) (NoValue, error) {
	return nil, HandlerErrorf(HandlerErrorTypeUnauthorized, "unauthorized in test")
}

func (h *authRejectionHandler) Cancel(ctx context.Context, token string, options CancelOperationOptions) error {
	return HandlerErrorf(HandlerErrorTypeUnauthorized, "unauthorized in test")
}

func (h *authRejectionHandler) GetInfo(ctx context.Context, token string, options GetOperationInfoOptions) (*OperationInfo, error) {
	return nil, HandlerErrorf(HandlerErrorTypeUnauthorized, "unauthorized in test")
}

func TestHandlerError(t *testing.T) {
	var handlerError *HandlerError

	registry := NewServiceRegistry()
	svc := NewService(testService)
	require.NoError(t, svc.Register(&authRejectionHandler{}))
	require.NoError(t, registry.Register(svc))

	handler, err := registry.NewHandler()
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	_, err = StartOperation(ctx, client, &authRejectionHandler{}, nil, StartOperationOptions{})
	require.ErrorAs(t, err, &handlerError)
	require.Equal(t, HandlerErrorTypeUnauthorized, handlerError.Type)
	require.Equal(t, "unauthorized in test", handlerError.Cause.Error())

	handle, err := NewOperationHandle(client, &authRejectionHandler{}, "dont-care")
	require.NoError(t, err)

	_, err = handle.GetInfo(ctx, GetOperationInfoOptions{})
	require.ErrorAs(t, err, &handlerError)
	require.Equal(t, HandlerErrorTypeUnauthorized, handlerError.Type)
	require.Equal(t, "unauthorized in test", handlerError.Cause.Error())

	err = handle.Cancel(ctx, CancelOperationOptions{})
	require.ErrorAs(t, err, &handlerError)
	require.Equal(t, HandlerErrorTypeUnauthorized, handlerError.Type)
	require.Equal(t, "unauthorized in test", handlerError.Cause.Error())

	_, err = handle.GetResult(ctx, GetOperationResultOptions{})
	require.ErrorAs(t, err, &handlerError)
	require.Equal(t, HandlerErrorTypeUnauthorized, handlerError.Type)
	require.Equal(t, "unauthorized in test", handlerError.Cause.Error())
}

func TestInputOutputType(t *testing.T) {
	require.True(t, reflect.TypeOf(3).AssignableTo(numberValidatorOperation.InputType()))
	require.False(t, reflect.TypeOf("s").AssignableTo(numberValidatorOperation.InputType()))

	require.True(t, reflect.TypeOf(3).AssignableTo(numberValidatorOperation.OutputType()))
	require.False(t, reflect.TypeOf("s").AssignableTo(numberValidatorOperation.OutputType()))
}

func TestOperationInterceptor(t *testing.T) {
	registry := NewServiceRegistry()
	svc := NewService(testService)
	require.NoError(t, svc.Register(
		asyncNumberValidatorOperationInstance,
	))

	var logs []string
	// Register the logging middleware after the auth middleware to ensure the auth middleware is called first.
	// any middleware that returns an error will prevent the operation from being called.
	registry.Use(newAuthMiddleware("auth-key"), newLoggingMiddleware(func(log string) {
		logs = append(logs, log)
	}))
	require.NoError(t, registry.Register(svc))

	handler, err := registry.NewHandler()
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	_, err = StartOperation(ctx, client, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{})
	require.ErrorContains(t, err, "unauthorized")

	authHeader := map[string]string{"authorization": "auth-key"}
	result, err := StartOperation(ctx, client, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{
		Header: authHeader,
	})
	require.NoError(t, err)
	require.ErrorContains(t, result.Async().Cancel(ctx, CancelOperationOptions{}), "unauthorized")
	require.NoError(t, result.Async().Cancel(ctx, CancelOperationOptions{Header: authHeader}))
	// Assert the logger  only contains calls from successful operations.
	require.Len(t, logs, 2)
	require.Contains(t, logs[0], "starting operation async-number-validator")
	require.Contains(t, logs[1], "cancel operation async-number-validator")
}

func newAuthMiddleware(authKey string) MiddlewareFunc {
	return func(ctx context.Context, next OperationHandler[any, any]) (OperationHandler[any, any], error) {
		info := ExtractHandlerInfo(ctx)
		if info.Header.Get("authorization") != authKey {
			return nil, HandlerErrorf(HandlerErrorTypeUnauthorized, "unauthorized")
		}
		return next, nil
	}
}

type loggingOperation struct {
	UnimplementedOperation[any, any]
	Operation OperationHandler[any, any]
	name      string
	output    func(string)
}

func (lo *loggingOperation) Start(ctx context.Context, input any, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	lo.output(fmt.Sprintf("starting operation %s", lo.name))
	return lo.Operation.Start(ctx, input, options)
}

func (lo *loggingOperation) GetResult(ctx context.Context, id string, options GetOperationResultOptions) (any, error) {
	lo.output(fmt.Sprintf("getting result for operation %s", lo.name))
	return lo.Operation.GetResult(ctx, id, options)
}

func (lo *loggingOperation) Cancel(ctx context.Context, id string, options CancelOperationOptions) error {
	lo.output(fmt.Sprintf("cancel operation %s", lo.name))
	return lo.Operation.Cancel(ctx, id, options)
}

func (lo *loggingOperation) GetInfo(ctx context.Context, id string, options GetOperationInfoOptions) (*OperationInfo, error) {
	lo.output(fmt.Sprintf("getting info for operation %s", lo.name))
	return lo.Operation.GetInfo(ctx, id, options)
}

func newLoggingMiddleware(output func(string)) MiddlewareFunc {
	return func(ctx context.Context, next OperationHandler[any, any]) (OperationHandler[any, any], error) {
		info := ExtractHandlerInfo(ctx)

		return &loggingOperation{
			Operation: next,
			name:      info.Operation,
			output:    output,
		}, nil
	}
}
