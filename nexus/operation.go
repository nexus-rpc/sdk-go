package nexus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type NoValue *struct{}

type Operation[I, O any] interface {
	Name() string
	// A type inference helper for implementions of this interface.
	inferType(I, O)
}

// TODO: document me
type OperationDefinition[I, O any] string

// Name implements Operation.
func (d OperationDefinition[I, O]) Name() string {
	return string(d)
}

// inferType implements Operation.
func (OperationDefinition[I, O]) inferType(I, O) {} //nolint:unused

var _ Operation[any, any] = OperationDefinition[any, any]("")

type UntypedOperationHandler interface {
	Name() string
	mustEmbedUnimplementedOperationHandler()
}

// OperationHandler is a handler for a single operation.
type OperationHandler[I, O any] interface {
	Operation[I, O]
	UntypedOperationHandler
	Start(context.Context, I, StartOperationOptions) (HandlerStartOperationResult[O], error)
	Cancel(context.Context, string, CancelOperationOptions) error
	GetResult(context.Context, string, GetOperationResultOptions) (O, error)
	GetInfo(context.Context, string, GetOperationInfoOptions) (*OperationInfo, error)
}

type syncOperationHandler[I, O any] struct {
	UnimplementedOperationHandler[I, O]

	Handler func(context.Context, I, StartOperationOptions) (O, error)
	name    string
}

func NewSyncOperation[I, O any](name string, handler func(context.Context, I, StartOperationOptions) (O, error)) OperationHandler[I, O] {
	return &syncOperationHandler[I, O]{
		name:    name,
		Handler: handler,
	}
}

// Name implements OperationHandler.
func (h *syncOperationHandler[I, O]) Name() string {
	return h.name
}

// StartOperation implements OperationHandler.
func (h *syncOperationHandler[I, O]) Start(ctx context.Context, input I, options StartOperationOptions) (HandlerStartOperationResult[O], error) {
	o, err := h.Handler(ctx, input, options)
	if err != nil {
		return nil, err
	}
	return &HandlerStartOperationResultSync[O]{o}, err
}

var _ OperationHandler[any, any] = &syncOperationHandler[any, any]{}

type OperationDirectoryHandlerOptions struct {
	Operations []UntypedOperationHandler
}

type OperationDirectoryHandler struct {
	UnimplementedServiceHandler

	operations map[string]UntypedOperationHandler
}

func NewOperationDirectoryHandler(options OperationDirectoryHandlerOptions) (*OperationDirectoryHandler, error) {
	mapped := make(map[string]UntypedOperationHandler, len(options.Operations))
	if len(options.Operations) == 0 {
		return nil, errors.New("must provide at least one operation")
	}
	dups := []string{}

	for _, op := range options.Operations {
		if _, found := mapped[op.Name()]; found {
			dups = append(dups, op.Name())
		}
		mapped[op.Name()] = op
	}
	if len(dups) > 0 {
		return nil, fmt.Errorf("duplicate operations: %s", strings.Join(dups, ", "))
	}
	return &OperationDirectoryHandler{operations: mapped}, nil
}

// CancelOperation implements Handler.
func (d *OperationDirectoryHandler) CancelOperation(ctx context.Context, operation string, operationID string, options CancelOperationOptions) error {
	h, ok := d.operations[operation]
	if !ok {
		return HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	m, _ := reflect.TypeOf(h).MethodByName("Cancel")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), reflect.ValueOf(operationID), reflect.ValueOf(options)})
	if values[0].IsNil() {
		return nil
	}
	return values[0].Interface().(error)
}

// GetOperationInfo implements Handler.
func (d *OperationDirectoryHandler) GetOperationInfo(ctx context.Context, operation string, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	h, ok := d.operations[operation]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	m, _ := reflect.TypeOf(h).MethodByName("GetInfo")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), reflect.ValueOf(operationID), reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret.(*OperationInfo), nil
}

// GetOperationResult implements Handler.
func (d *OperationDirectoryHandler) GetOperationResult(ctx context.Context, operation string, operationID string, options GetOperationResultOptions) (any, error) {
	h, ok := d.operations[operation]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	m, _ := reflect.TypeOf(h).MethodByName("GetResult")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), reflect.ValueOf(operationID), reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret, nil
}

// StartOperation implements Handler.
func (d *OperationDirectoryHandler) StartOperation(ctx context.Context, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	h, ok := d.operations[operation]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	m, _ := reflect.TypeOf(h).MethodByName("Start")
	inputType := m.Type.In(2)
	iptr := reflect.New(inputType).Interface()
	if err := input.Consume(iptr); err != nil {
		// TODO: log the error?
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid input")
	}
	i := reflect.ValueOf(iptr).Elem()

	values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), i, reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret.(HandlerStartOperationResult[any]), nil
}

var _ ServiceHandler = &OperationDirectoryHandler{}

func ExecuteOperation[I, O any](ctx context.Context, client *Client, operation Operation[I, O], input I, request ExecuteOperationOptions) (O, error) {
	var o O
	value, err := client.ExecuteOperation(ctx, operation.Name(), input, request)
	if err != nil {
		return o, err
	}
	return o, value.Consume(&o)
}

func StartOperation[I, O any](ctx context.Context, client *Client, operation Operation[I, O], input I, request StartOperationOptions) (*ClientStartOperationResult[O], error) {
	result, err := client.StartOperation(ctx, operation.Name(), input, request)
	if err != nil {
		return nil, err
	}
	if result.Successful != nil {
		var o O
		if err := result.Successful.Consume(&o); err != nil {
			return nil, err
		}
		return &ClientStartOperationResult[O]{Successful: o}, nil
	}
	handle := OperationHandle[O]{client: client, Operation: operation.Name(), ID: result.Pending.ID}
	return &ClientStartOperationResult[O]{Pending: &handle}, nil
}

func NewHandle[T any](client *Client, operation, operationID string) (*OperationHandle[T], error) {
	var es []error
	if operation == "" {
		es = append(es, errEmptyOperationName)
	}
	if operationID == "" {
		es = append(es, errEmptyOperationID)
	}
	if len(es) > 0 {
		return nil, errors.Join(es...)
	}
	return &OperationHandle[T]{client: client, Operation: operation, ID: operationID}, nil
}
