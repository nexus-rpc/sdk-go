package nexus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	return &HandlerStartOperationResultAsync{OperationToken: fmt.Sprintf("%d", input)}, nil
}

func (h *asyncNumberValidatorOperation) Cancel(ctx context.Context, token string, options CancelOperationOptions) error {
	if token != "token" {
		return fmt.Errorf(`invalid token: %q, expected: "token"`, token)
	}
	return nil
}

var asyncNumberValidatorOperationInstance = &asyncNumberValidatorOperation{}

func TestRegistrationErrors(t *testing.T) {
	reg := NewServiceRegistry()
	svc := NewService("service")
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
	require.ErrorContains(t, err, fmt.Sprintf("service %q has no operations registered", "service"))
}

func lv(t *testing.T, v any) *LazyValue {
	t.Helper()
	content, err := defaultSerializer.Serialize(v)
	require.NoError(t, err)
	return NewLazyValue(defaultSerializer, &Reader{
		ReadCloser: io.NopCloser(bytes.NewBuffer(content.Data)),
		Header:     content.Header,
	})
}

func startOperation(
	t *testing.T,
	handler Handler,
	svc *Service,
	op RegisterableOperation,
	input any,
	options StartOperationOptions,
) (HandlerStartOperationResult[any], error) {
	t.Helper()
	ctx := WithHandlerContext(context.Background(), HandlerInfo{
		Service:   svc.Name,
		Operation: op.Name(),
		Header:    options.Header,
	})
	return handler.StartOperation(ctx, svc.Name, op.Name(), lv(t, input), options)
}

func cancelOperation(
	t *testing.T,
	handler Handler,
	svc *Service,
	op RegisterableOperation,
	token string,
	options CancelOperationOptions,
) error {
	t.Helper()
	ctx := WithHandlerContext(context.Background(), HandlerInfo{
		Service:   svc.Name,
		Operation: op.Name(),
		Header:    options.Header,
	})
	return handler.CancelOperation(ctx, svc.Name, asyncNumberValidatorOperationInstance.Name(), token, options)
}

func TestStartOperation(t *testing.T) {
	registry := NewServiceRegistry()
	svc := NewService("service")
	require.NoError(t, svc.Register(
		numberValidatorOperation,
		asyncNumberValidatorOperationInstance,
	))
	require.NoError(t, registry.Register(svc))

	handler, err := registry.NewHandler()
	require.NoError(t, err)

	result, err := startOperation(t, handler, svc, numberValidatorOperation, 3, StartOperationOptions{})
	require.NoError(t, err)
	syncRes, ok := result.(*HandlerStartOperationResultSync[int])
	require.True(t, ok)
	require.Equal(t, 3, syncRes.Value)

	result, err = startOperation(t, handler, svc, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{})
	require.NoError(t, err)
	asyncRes, ok := result.(*HandlerStartOperationResultAsync)
	require.True(t, ok)
	require.Equal(t, "3", asyncRes.OperationToken)
}

func TestCancelOperation(t *testing.T) {
	registry := NewServiceRegistry()
	svc := NewService("service")
	require.NoError(t, svc.Register(
		asyncNumberValidatorOperationInstance,
	))
	require.NoError(t, registry.Register(svc))

	handler, err := registry.NewHandler()
	require.NoError(t, err)

	err = cancelOperation(t, handler, svc, asyncNumberValidatorOperationInstance, "token", CancelOperationOptions{})
	require.NoError(t, err)
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

func (h *authRejectionHandler) Cancel(ctx context.Context, token string, options CancelOperationOptions) error {
	return HandlerErrorf(HandlerErrorTypeUnauthorized, "unauthorized in test")
}

func TestHandlerError(t *testing.T) {

	registry := NewServiceRegistry()
	svc := NewService("service")
	authRejectionHandlerInstance := &authRejectionHandler{}
	require.NoError(t, svc.Register(authRejectionHandlerInstance))
	require.NoError(t, registry.Register(svc))

	handler, err := registry.NewHandler()
	require.NoError(t, err)

	var handlerError *HandlerError
	_, err = startOperation(t, handler, svc, authRejectionHandlerInstance, nil, StartOperationOptions{})
	require.ErrorAs(t, err, &handlerError)
	require.Equal(t, HandlerErrorTypeUnauthorized, handlerError.Type)
	require.Equal(t, "unauthorized in test", handlerError.Message)

	err = cancelOperation(t, handler, svc, asyncNumberValidatorOperationInstance, "token", CancelOperationOptions{})
	require.ErrorAs(t, err, &handlerError)
	require.Equal(t, HandlerErrorTypeUnauthorized, handlerError.Type)
	require.Equal(t, "unauthorized in test", handlerError.Message)
}

func TestInputOutputType(t *testing.T) {
	require.True(t, reflect.TypeOf(3).AssignableTo(numberValidatorOperation.InputType()))
	require.False(t, reflect.TypeOf("s").AssignableTo(numberValidatorOperation.InputType()))

	require.True(t, reflect.TypeOf(3).AssignableTo(numberValidatorOperation.OutputType()))
	require.False(t, reflect.TypeOf("s").AssignableTo(numberValidatorOperation.OutputType()))
}

func TestOperationInterceptor(t *testing.T) {
	registry := NewServiceRegistry()
	svc := NewService("service")
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

	_, err = startOperation(t, handler, svc, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{})
	require.ErrorContains(t, err, "unauthorized")

	authHeader := map[string]string{"authorization": "auth-key"}
	_, err = startOperation(t, handler, svc, asyncNumberValidatorOperationInstance, 3, StartOperationOptions{
		Header: authHeader,
	})
	require.NoError(t, err)
	require.ErrorContains(t, cancelOperation(t, handler, svc, asyncNumberValidatorOperationInstance, "token", CancelOperationOptions{}), "unauthorized")
	require.NoError(t, cancelOperation(t, handler, svc, asyncNumberValidatorOperationInstance, "token", CancelOperationOptions{Header: authHeader}))
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

func (lo *loggingOperation) Cancel(ctx context.Context, id string, options CancelOperationOptions) error {
	lo.output(fmt.Sprintf("cancel operation %s", lo.name))
	return lo.Operation.Cancel(ctx, id, options)
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
