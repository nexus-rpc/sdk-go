package nexus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type dynamicEcho struct {
	UnimplementedOperation[*LazyValue, any]
	cancelled atomic.Int32
}

func (d *dynamicEcho) Start(ctx context.Context, input *LazyValue, _ StartOperationOptions) (HandlerStartOperationResult[any], error) {
	var value int
	if err := input.Consume(&value); err != nil {
		return nil, err
	}
	return &HandlerStartOperationResultAsync{OperationToken: fmt.Sprintf("%s:%d", ExtractHandlerInfo(ctx).Operation, value)}, nil
}

func (d *dynamicEcho) Cancel(context.Context, string, CancelOperationOptions) error {
	d.cancelled.Add(1)
	return nil
}

type namedCancelOperation struct {
	UnimplementedOperation[int, string]
	cancelled atomic.Int32
}

func (*namedCancelOperation) Name() string { return "named" }
func (n *namedCancelOperation) Start(context.Context, int, StartOperationOptions) (HandlerStartOperationResult[string], error) {
	return &HandlerStartOperationResultSync[string]{Value: "static"}, nil
}

func (n *namedCancelOperation) Cancel(context.Context, string, CancelOperationOptions) error {
	n.cancelled.Add(1)
	return nil
}

// TestRegisterDynamicOperation checks that invalid registrations leave the first handler registered intact.
func TestRegisterDynamicOperation(t *testing.T) {
	service := NewService("tools")
	handler := &dynamicEcho{}
	t.Run("nil handler registration fails", func(t *testing.T) {
		require.EqualError(t, service.RegisterDynamicOperation(nil), "dynamic operation handler must not be nil")
	})

	t.Run("second registration fails", func(t *testing.T) {
		require.NoError(t, service.RegisterDynamicOperation(handler))
		require.EqualError(t, service.RegisterDynamicOperation(handler), "dynamic operation handler already registered")
		// Check that the first handler is still registered.
		require.Equal(t, handler, service.DynamicOperation())
	})

	t.Run("cannot overwrite with nil", func(t *testing.T) {
		require.EqualError(t, service.RegisterDynamicOperation(nil), "dynamic operation handler must not be nil")
		// Check that the first handler is still registered.
		require.Equal(t, handler, service.DynamicOperation())
	})
}

// TestDynamicOperationRouting checks routing behavior:
// - Named operations are always routed to the named handler.
// - Unmatched operations are routed to the dynamic handler if present.
// - Unmatched operations are rejected if no dynamic handler is present.
func TestDynamicOperationRouting(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		service                  string
		operation                string
		serviceHasDynamicHandler bool
		wantNotFound             bool
	}{
		{name: "named with fallback", service: "tools", operation: "named", serviceHasDynamicHandler: true},
		{name: "named without fallback", service: "tools", operation: "named"},
		{name: "unmatched with fallback", service: "tools", operation: "echo", serviceHasDynamicHandler: true},
		{name: "unmatched without fallback", service: "tools", operation: "echo", wantNotFound: true},
		{name: "unknown service", service: "missing", operation: "echo", serviceHasDynamicHandler: true, wantNotFound: true},
	} {
		for _, method := range []string{"start", "cancel"} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				// Setup service and registry.
				service := NewService("tools")
				dynamic, named := &dynamicEcho{}, &namedCancelOperation{}
				if tc.serviceHasDynamicHandler {
					require.NoError(t, service.RegisterDynamicOperation(dynamic))
				}
				service.MustRegister(named)
				registry := NewServiceRegistry()
				registry.MustRegister(service)
				handler, err := registry.NewHandler()
				require.NoError(t, err)

				ctx := WithHandlerContext(context.Background(), HandlerInfo{Service: tc.service, Operation: tc.operation})
				if method == "start" {
					input := 5
					var result HandlerStartOperationResult[any]
					result, err = handler.StartOperation(ctx, tc.service, tc.operation, lv(t, input), StartOperationOptions{})
					if !tc.wantNotFound {
						require.NoError(t, err)
						if tc.operation == "named" {
							require.Equal(t, "static", result.(*HandlerStartOperationResultSync[string]).Value)
						} else {
							require.Equal(t, fmt.Sprintf("%s:%d", tc.operation, input), result.(*HandlerStartOperationResultAsync).OperationToken)
						}
					}
				} else {
					err = handler.CancelOperation(ctx, tc.service, tc.operation, "token", CancelOperationOptions{})
					if !tc.wantNotFound {
						require.NoError(t, err)
						if tc.operation == "named" {
							require.EqualValues(t, 1, named.cancelled.Load())
							require.Zero(t, dynamic.cancelled.Load())
						} else {
							require.EqualValues(t, 1, dynamic.cancelled.Load())
							require.Zero(t, named.cancelled.Load())
						}
					}
				}
				if tc.wantNotFound {
					var handlerErr *HandlerError
					require.ErrorAs(t, err, &handlerErr)
					require.Equal(t, HandlerErrorTypeNotFound, handlerErr.Type)
					require.Zero(t, named.cancelled.Load())
					require.Zero(t, dynamic.cancelled.Load())
				}
				require.Nil(t, service.Operation("echo"))
			})
		}
	}
}

// TestConcurrentDynamicOperationDispatch checks that concurrent requests keep their operation names and inputs separate.
func TestConcurrentDynamicOperationDispatch(t *testing.T) {
	service := NewService("tools")
	require.NoError(t, service.RegisterDynamicOperation(&dynamicEcho{}))
	registry := NewServiceRegistry()
	registry.MustRegister(service)
	handler, err := registry.NewHandler()
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			operation := fmt.Sprintf("echo-%d", i)
			ctx := WithHandlerContext(context.Background(), HandlerInfo{Service: "tools", Operation: operation})
			for j := 0; j < 100; j++ {
				result, err := handler.StartOperation(ctx, "tools", operation, lv(t, j), StartOperationOptions{})
				if err != nil {
					t.Error(err)
					return
				}
				if result.(*HandlerStartOperationResultAsync).OperationToken != fmt.Sprintf("%s:%d", operation, j) {
					t.Error("wrong dynamic result")
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

// TestDynamicServiceValidation ensures that we enforce a service needs:
// - at least one named operation, or
// - a dynamic handler
// to be valid.
func TestDynamicServiceValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		named   bool
		dynamic bool
	}{
		{name: "empty"},
		{name: "named only", named: true},
		{name: "dynamic only", dynamic: true},
		{name: "named and dynamic", named: true, dynamic: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService("tools")
			if tc.named {
				service.MustRegister(&namedCancelOperation{})
			}
			if tc.dynamic {
				require.NoError(t, service.RegisterDynamicOperation(&dynamicEcho{}))
			}
			registry := NewServiceRegistry()
			registry.MustRegister(service)
			_, err := registry.NewHandler()
			if !tc.named && !tc.dynamic {
				require.ErrorContains(t, err, "has no operations registered")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type dynamicSync struct {
	UnimplementedOperation[*LazyValue, any]
}

func (*dynamicSync) Start(ctx context.Context, input *LazyValue, _ StartOperationOptions) (HandlerStartOperationResult[any], error) {
	var value string
	if err := input.Consume(&value); err != nil {
		return nil, err
	}
	return &HandlerStartOperationResultSync[any]{Value: ExtractHandlerInfo(ctx).Operation + ":" + value}, nil
}

// TestDynamicOperationSyncResult checks that the fallback can complete synchronously after decoding raw input.
func TestDynamicOperationSyncResult(t *testing.T) {
	service := NewService("tools")
	require.NoError(t, service.RegisterDynamicOperation(&dynamicSync{}))
	registry := NewServiceRegistry()
	registry.MustRegister(service)
	handler, err := registry.NewHandler()
	require.NoError(t, err)
	ctx := WithHandlerContext(context.Background(), HandlerInfo{Service: "tools", Operation: "echo"})
	result, err := handler.StartOperation(ctx, "tools", "echo", lv(t, "in"), StartOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, "echo:in", result.(*HandlerStartOperationResultSync[any]).Value)
}

type dynamicNotFound struct {
	UnimplementedOperation[*LazyValue, any]
	calls atomic.Int32
}

func (d *dynamicNotFound) Start(context.Context, *LazyValue, StartOperationOptions) (HandlerStartOperationResult[any], error) {
	d.calls.Add(1)
	return nil, NewHandlerErrorf(HandlerErrorTypeNotFound, "unknown tool")
}

func (d *dynamicNotFound) Cancel(context.Context, string, CancelOperationOptions) error {
	d.calls.Add(1)
	return NewHandlerErrorf(HandlerErrorTypeNotFound, "unknown tool")
}

type namedNotFound struct {
	UnimplementedOperation[int, any]
}

func (*namedNotFound) Name() string { return "reject" }

func (*namedNotFound) Start(context.Context, int, StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return nil, NewHandlerErrorf(HandlerErrorTypeNotFound, "named rejection")
}

func (*namedNotFound) Cancel(context.Context, string, CancelOperationOptions) error {
	return NewHandlerErrorf(HandlerErrorTypeNotFound, "named rejection")
}

// TestNamedOperationNotFoundDoesNotUseFallback checks that if a named handler returns HandlerErrorTypeNotFound,
// we don't trigger dynamic handler routing.
func TestNamedOperationNotFoundDoesNotUseFallback(t *testing.T) {
	for _, method := range []string{"start", "cancel"} {
		t.Run(method, func(t *testing.T) {
			service := NewService("tools")
			dynamic := &dynamicEcho{}
			service.MustRegister(&namedNotFound{})
			require.NoError(t, service.RegisterDynamicOperation(dynamic))
			registry := NewServiceRegistry()
			registry.MustRegister(service)
			handler, err := registry.NewHandler()
			require.NoError(t, err)
			ctx := WithHandlerContext(context.Background(), HandlerInfo{Service: "tools", Operation: "reject"})
			if method == "start" {
				_, err = handler.StartOperation(ctx, "tools", "reject", lv(t, 1), StartOperationOptions{})
			} else {
				err = handler.CancelOperation(ctx, "tools", "reject", "token", CancelOperationOptions{})
			}
			var handlerErr *HandlerError
			require.ErrorAs(t, err, &handlerErr)
			require.Equal(t, HandlerErrorTypeNotFound, handlerErr.Type)
			require.Equal(t, "named rejection", handlerErr.Message)
			require.Zero(t, dynamic.cancelled.Load())
		})
	}
}

// TestDynamicOperationNotFoundIsFinal checks that if a dynamic handler returns HandlerErrorTypeNotFound,
// we keep that error unchanged.
func TestDynamicOperationNotFoundIsFinal(t *testing.T) {
	for _, method := range []string{"start", "cancel"} {
		t.Run(method, func(t *testing.T) {
			service := NewService("tools")
			dynamic := &dynamicNotFound{}
			require.NoError(t, service.RegisterDynamicOperation(dynamic))
			registry := NewServiceRegistry()
			registry.MustRegister(service)
			handler, err := registry.NewHandler()
			require.NoError(t, err)
			ctx := WithHandlerContext(context.Background(), HandlerInfo{Service: "tools", Operation: "unknown"})
			if method == "start" {
				_, err = handler.StartOperation(ctx, "tools", "unknown", lv(t, 1), StartOperationOptions{})
			} else {
				err = handler.CancelOperation(ctx, "tools", "unknown", "token", CancelOperationOptions{})
			}
			var handlerErr *HandlerError
			require.ErrorAs(t, err, &handlerErr)
			require.Equal(t, HandlerErrorTypeNotFound, handlerErr.Type)
			require.Equal(t, "unknown tool", handlerErr.Message)
			require.EqualValues(t, 1, dynamic.calls.Load())
		})
	}
}

// TestDynamicOperationMiddleware checks that start middleware sees raw input and cancel uses the same middleware chain.
func TestDynamicOperationMiddleware(t *testing.T) {
	service := NewService("tools")
	dynamic := &dynamicEcho{}
	require.NoError(t, service.RegisterDynamicOperation(dynamic))
	registry := NewServiceRegistry()
	registry.MustRegister(service)
	var logs []string
	registry.Use(newLoggingMiddleware(func(s string) { logs = append(logs, s) }))
	var inputs []any
	registry.Use(func(ctx context.Context, next OperationHandler[any, any]) (OperationHandler[any, any], error) {
		return &inputRecorder{OperationHandler: next, inputs: &inputs}, nil
	})
	handler, err := registry.NewHandler()
	require.NoError(t, err)
	ctx := WithHandlerContext(context.Background(), HandlerInfo{Service: "tools", Operation: "echo"})
	_, err = handler.StartOperation(ctx, "tools", "echo", lv(t, 3), StartOperationOptions{})
	require.NoError(t, err)
	require.NoError(t, handler.CancelOperation(ctx, "tools", "echo", "echo:3", CancelOperationOptions{}))
	require.Equal(t, []string{"starting operation echo", "cancel operation echo"}, logs)
	require.Len(t, inputs, 1)
	require.IsType(t, &LazyValue{}, inputs[0])
	require.EqualValues(t, 1, dynamic.cancelled.Load())
}

type inputRecorder struct {
	OperationHandler[any, any]
	inputs *[]any
}

func (r *inputRecorder) Start(ctx context.Context, input any, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	*r.inputs = append(*r.inputs, input)
	return r.OperationHandler.Start(ctx, input, options)
}
