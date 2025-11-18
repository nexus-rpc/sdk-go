# Nexus Go SDK

[![PkgGoDev](https://pkg.go.dev/badge/github.com/nexus-rpc/sdk-go)](https://pkg.go.dev/github.com/nexus-rpc/sdk-go)
[![Continuous Integration](https://github.com/nexus-rpc/sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/nexus-rpc/sdk-go/actions/workflows/ci.yml)

Go SDK for modelling applications to work with the [Nexus RPC](https://github.com/nexus-rpc/api) specification.
**Client and server implementation not included**.

## What is Nexus?

Nexus is an RPC protocol for expressing arbitrary duration operations.

A Nexus **caller** calls a **handler**. The handler may respond inline (synchronous response) or return a token
referencing the ongoing operation (asynchronous response), which the caller may use to cancel the operation. In lieu of
a higher level service contract, the caller cannot determine whether an operation is going to resolve synchronously or
asynchronously, and should specify a callback URL, which the handler uses to deliver the result of an asynchronous
operation when it is ready.

## Installation

```shell
go get -u github.com/nexus-rpc/sdk-go
```

## Usage

### Import

```go
import (
	"github.com/nexus-rpc/sdk-go/nexus"
)
```

### Define an Operation Reference

Operation references can be used in client implementations for type safe invocation without requiring access to a
concrete implementation.

```go
var syncOperationRef = nexus.NewOperationReference[MyInput, MyOutput]("sync-example")
```

### Implement a Sync Operation

```go
var syncOperation = nexus.NewSyncOperation("sync-example", func(ctx context.Context, input MyInput, options StartOperationOptions) (MyOutput, error) {
	return MyOutput{Field: "value"}, nil
})
```

### Implement an Async Operation

```go
type myAsyncOperation struct {
	nexus.UnimplementedOperation[MyInput, MyOutput]
}

func (h *myAsyncOperation) Name() string {
	return "example"
}

func (h *myAsyncOperation) Start(ctx context.Context, input MyInput, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[MyOutput], error) {
	// alternatively return &nexus.HandlerStartOperationResultSync{Value: MyOutput{}}, nil
	return &nexus.HandlerStartOperationResultAsync{OperationToken: "BASE64_ENCODED_DATA"}, nil
}

func (h *myAsyncOperation) Cancel(ctx context.Context, token string, options nexus.CancelOperationOptions) error {
	fmt.Println("Canceling", h.Name(), "with token:", token)
	return nil
}
```

### Register an Operation implementation with a Service

```go
var service = nexus.NewService("example-service")
svc.MustRegister(operation, &myAsyncOperation{})
```

### Resolve an operation as failed

```go
func (h *myAsyncOperation) Start(ctx context.Context, input MyInput, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[MyOutput], error) {
	// Alternatively use NewOperationCanceledError to resolve an operation as canceled.
	return nil, nexus.NewOperationFailedError("do or do not, there is not try")
}
```

### Fail any handler method (Start or Cancel)

Returning an arbitrary error from any of the `Operation` and `OperationHandler` methods will result in the error being
logged and the request responded to with a generic Internal Server Error and Failure message.

To fail a request with a custom status error type and failure message, return a `nexus.HandlerError` as the error.
The error can either be constructed directly or with the `HandlerErrorf` helper.

```go
func (h *myAsyncOperation) Start(ctx context.Context, input MyInput, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[MyOutput], error) {
	return nil, nexus.HandlerErrorf(nexus.HandlerErrorTypeBadRequest, "invalid input field: %v", input.Field)
}
```

## Failure Structs

`nexus` exports a `Failure` struct that is used in both the client and handlers to represent both application level
operation failures and framework level HTTP request errors.

`Failure`s typically contain a single `Message` string but may also convey arbitrary JSONable `Details` and `Metadata`.

The `Details` field is encoded and it is up to the library user to encode to and decode from it.

A failure can be either directly attached to `HandlerError` and `OperationError` instances by providing `FailureError`
as the `Cause`, or indirectly by implementing the `FailureConverter` interface, which can translate arbitrary user
defined errors to `Failure` instances and back.

### Links

Nexus operations can bi-directionally link the caller and handler for tracing the execution path. A caller may provide
a set of `Link` objects via `StartOperationOptions` that the handler may log or attach to any underlying resources
backing the operation. A handler may attach backlinks when responding to a `StartOperation` request via the a
`AddHandlerLinks` method.

#### Handler

```go
func (h *myArbitraryLengthOperation) Start(ctx context.Context, input MyInput, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[MyOutput], error) {
	output, backlinks, _ := createMyBackingResourceAndAttachCallerLinks(ctx, input, options.Links)
	nexus.AddHandlerLinks(ctx, backlinks)
	return output, nil
}

result, _ := nexus.StartOperation(ctx, client, operation, MyInput{Field: "value"}, nexus.StartOperationOptions{
	Links: []nexus.Link{
		{
			Type: "org.my.MyResource",
			URL:  &url.URL{/* ... */},
		},
	},
})
fmt.Println("got result with backlinks", result.Links)
```

### Middleware

The ServiceRegistry supports middleware registration via the `Use` method. The registry's handler will invoke every
registered middleware in registration order. Typical use cases for middleware include global enforcement of
authorization and logging.

Middleware is implemented as a function that takes the current context and the next handler in the invocation chain and
returns a new handler to invoke. The function can pass through the given handler or return an error to abort the
execution. The registered middleware function has access to common handler information such as the current service,
operation, and request headers. To get access to more specific handler method information, such as inputs and operation
tokens, wrap the given handler.

**Example**

```go
type loggingOperation struct {
	nexus.UnimplementedOperation[any, any] // All OperationHandlers must embed this.
	next nexus.OperationHandler[any, any]
}

func (lo *loggingOperation) Start(ctx context.Context, input any, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[any], error) {
	log.Println("starting operation", nexus.ExtractHandlerInfo(ctx).Operation)
	return lo.next.Start(ctx, input, options)
}

func (lo *loggingOperation) Cancel(ctx context.Context, token string, options nexus.CancelOperationOptions) error {
	log.Printf("canceling operation", nexus.ExtractHandlerInfo(ctx).Operation)
	return lo.next.Cancel(ctx, token, options)
}

registry.Use(func(ctx context.Context, next nexus.OperationHandler[any, any]) (nexus.OperationHandler[any, any], error) {
	// Optionally call nexus.ExtractHandlerInfo(ctx) here.
	return &loggingOperation{next: next}, nil
})
```

## Contributing

### Prerequisites

- [Go 1.25](https://go.dev/doc/install)
- [golangci-lint](https://golangci-lint.run/usage/install/)

### Test

```shell
go test -v ./...
```

### Lint

```shell
golangci-lint run --verbose --timeout 1m --fix=false
```
