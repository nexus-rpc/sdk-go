# Nexus Go SDK

[![PkgGoDev](https://pkg.go.dev/badge/github.com/nexus-rpc/sdk-go)](https://pkg.go.dev/github.com/nexus-rpc/sdk-go)
[![Continuous Integration](https://github.com/nexus-rpc/sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/nexus-rpc/sdk-go/actions/workflows/ci.yml)

Client and server package for working with the Nexus [HTTP API](https://github.com/nexus-rpc/api).

**⚠️ EXPERIMENTAL ⚠️**

## What is Nexus?

Nexus is a synchronous RPC protocol. Arbitrary length operations are modelled on top of a set of pre-defined synchronous
RPCs.

A Nexus caller calls a handler. The handler may respond inline or return a reference for a future, asynchronous
operation. The caller can cancel an asynchronous operation, check for its outcome, or fetch its current state. The
caller can also specify a callback URL, which the handler uses to asynchronously deliver the result of an operation when
it is ready.

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

### Client

The Nexus HTTPClient is used to start operations and get [handles](#operationhandle) to existing, asynchronous operations.

#### Create an HTTPClient

```go
client, err := nexus.NewHTTPClient(nexus.HTTPClientOptions{
	BaseURL: "https://example.com/path/to/my/services",
	Service: "example-service",
})
```

#### Start an Operation

An OperationReference can be used to invoke an opertion in a typed way:

```go
// Create an operation reference for typed invocation.
// You may also use any Operation implementation for invocation (more on that below).
operation := nexus.NewOperationReference[MyInput, MyOutput]("example")

// StartOperationOptions can be used to explicitly set a request ID, headers, and callback URL.
result, err := nexus.StartOperation(ctx, client, operation, MyInput{Field: "value"}, nexus.StartOperationOptions{})
if err != nil {
	var unsuccessfulOperationError *nexus.UnsuccessfulOperationError
	if errors.As(err, &unsuccessfulOperationError) { // operation failed or canceled
		fmt.Printf("Operation unsuccessful with state: %s, failure message: %s\n", unsuccessfulOperationError.State, unsuccessfulOperationError.Failure.Message)
	}
	var handlerError *nexus.HandlerError
	if errors.As(err, &handlerError) {
		fmt.Printf("Handler returned an error, type: %s, failure message: %s\n", handlerError.Type, handlerError.Failure.Message)
	}
	// most other errors should be returned as *nexus.UnexpectedResponseError
}
if result.Successful != nil { // operation successful
	output := result.Successful // output is of type MyOutput
	fmt.Printf("Operation succeeded synchronously: %v\n", output)
} else { // operation started asynchronously
	handle := result.Pending
	fmt.Printf("Started asynchronous operation with ID: %s\n", handle.ID)
}
```

Alternatively, an operation can be started by name:

```go
result, err := client.StartOperation(ctx, "example", MyInput{Field: "value"}, nexus.StartOperationOptions{})
// result.Succesful is a LazyValue that must be consumed to free up the underlying connection.
```

#### Start an Operation and Await its Completion

The HTTPClient provides the `ExecuteOperation` helper function as a shorthand for `StartOperation` and issuing a `GetResult`
in case the operation is asynchronous.

```go
// By default ExecuteOperation will long poll until the context deadline for the operation to complete.
// Set ExecuteOperationOptions.Wait to change the wait duration.
output, err := nexus.ExecuteOperation(ctx, client, operation, MyInput{}, nexus.ExecuteOperationOptions{})
if err != nil {
	// handle nexus.UnsuccessfulOperationError, nexus.ErrOperationStillRunning and, context.DeadlineExceeded
}
fmt.Printf("Operation succeeded: %v\n", output) // output is of type MyOutput
```

Alternatively, an operation can be executed by name:

```
lazyValue, err := client.ExecuteOperation(ctx, "example", MyInput{}, nexus.ExecuteOperationOptions{})
// lazyValue that must be consumed to free up the underlying connection.
```

#### Get a Handle to an Existing Operation

Getting a handle does not incur a trip to the server.

```go
// Get a handle from an OperationReference
handle, _ := nexus.NewHandle(client, operation, "operation ID")

// Get a handle from a string name
handle, _ := client.NewHandle("operation name", "operation ID")
```

### OperationHandle

`OperationHandle`s are used to cancel and get the result and status of an operation.

Handles expose a couple of readonly attributes: `Operation` and `ID`.

#### Operation

`Operation` is the name of the operation this handle represents.

#### ID

`ID` is the operation ID as returned by a Nexus handler in the response to `StartOperation` or set by the client in the
`NewHandle` method.

#### Get the Result of an Operation

The `GetResult` method is used to get the result of an operation, issuing a network request to the handle's client's
configured endpoint.

By default, GetResult returns (nil, `ErrOperationStillRunning`) immediately after issuing a call if the operation has
not yet completed.

Callers may set GetOperationResultOptions.Wait to a value greater than 0 to alter this behavior, causing the client to
long poll for the result issuing one or more requests until the provided wait period exceeds, in which case (nil,
`ErrOperationStillRunning`) is returned.

The wait time is capped to the deadline of the provided context. Make sure to handle both context deadline errors and
`ErrOperationStillRunning`.

Note that the wait period is enforced by the server and may not be respected if the server is misbehaving. Set the
context deadline to the max allowed wait period to ensure this call returns in a timely fashion.

Custom request headers may be provided via `GetOperationResultOptions`.

When a handle is created from an OperationReference, `GetResult` returns a result of the reference's output type. When a
handle is created from a name, `GetResult` returns a `LazyValue` which must be `Consume`d to free up the underlying
connection.

```go
result, err := handle.GetResult(ctx, nexus.GetOperationResultOptions{})
if err != nil {
	// handle nexus.UnsuccessfulOperationError, nexus.ErrOperationStillRunning and, context.DeadlineExceeded
}
// result's type is the Handle's generic type T.
```

#### Get Operation Information

The `GetInfo` method is used to get operation information (currently only the operation's state) issuing a network
request to the service handler.

Custom request headers may be provided via `GetOperationInfoOptions`.

```go
info, _ := handle.GetInfo(ctx, nexus.GetOperationInfoOptions{})
```

#### Cancel an Operation

The `Cancel` method requests cancelation of an asynchronous operation.

Cancelation in Nexus is asynchronous and may be not be respected by the operation's implementation.

Custom request headers may be provided via `CancelOperationOptions`.

```go
_ := handle.Cancel(ctx, nexus.CancelOperationOptions{})
```

#### Complete an Operation

Handlers starting asynchronous operations may need to deliver responses via a caller specified callback URL.

`NewCompletionHTTPRequest` is used to construct an HTTP request to deliver operation completions - successful or
unsuccessful - to the provided callback URL.

To deliver successful completions, pass a `OperationCompletionSuccessful` struct pointer, which may also be constructed
with the `NewOperationCompletionSuccessful` helper.

Custom HTTP headers may be provided via `OperationCompletionSuccessful.Header`.

```go
completion, _ := nexus.NewOperationCompletionSuccessful(MyStruct{Field: "value"}, OperationCompletionSuccessfulOptions{})
request, _ := nexus.NewCompletionHTTPRequest(ctx, callbackURL, completion)
response, _ := http.DefaultClient.Do(request)
defer response.Body.Close()
_, err = io.ReadAll(response.Body)
fmt.Println("delivered completion with status code", response.StatusCode)
```

To deliver failed and canceled completions, pass a `OperationCompletionUnsuccessful` struct pointer with the failure and
state attached.

Custom HTTP headers may be provided via `OperationCompletionUnsuccessful.Header`.

```go
completion := &OperationCompletionUnsuccessful{
	State: nexus.OperationStateCanceled,
	Failure: &nexus.Failure{Message: "canceled as requested"},
}
request, _ := nexus.NewCompletionHTTPRequest(ctx, callbackURL, completion)
// ...
```

### Server

To handle operation requests, implement the `Operation` interface and use the `OperationRegistry` to create a `Handler`
that can be used to serve requests over HTTP.

Implement `CompletionHandler` to handle async delivery of operation completions.

#### Implement a Sync Operation

```go
var exampleOperation = NewSyncOperation("example", func(ctx context.Context, input MyInput, options StartOperationOptions) (MyOutput, error) {
	return MyOutput{Field: "value"}, nil
})
```

#### Implement an Arbitrary Length Operation

```go
type myArbitraryLengthOperation struct {
	nexus.UnimplementedOperation[MyInput, MyOutput]
}

func (h *myArbitraryLengthOperation) Name() string {
	return "alo-example"
}

func (h *myArbitraryLengthOperation) Start(ctx context.Context, input MyInput, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[MyOutput], error) {
	// alternatively return &HandlerStartOperationResultSync{Value: MyOutput{}}, nil
	return &HandlerStartOperationResultAsync{OperationID: "some-meaningful-id"}, nil
}

func (h *myArbitraryLengthOperation) GetResult(ctx context.Context, id string, options nexus.GetOperationResultOptions) (MyOutput, error) {
	return MyOutput{}, nil
}

func (h *myArbitraryLengthOperation) Cancel(ctx context.Context, id string, options nexus.CancelOperationOptions) error {
	fmt.Println("Canceling", h.Name(), "with ID:", request.OperationID)
	return nil
}

func (h *myArbitraryLengthOperation) GetInfo(ctx context.Context, id string, options nexus.GetOperationInfoOptions) (*nexus.OperationInfo, error) {
	return &nexus.OperationInfo{ID: id, State: nexus.OperationStateRunning}, nil
}
```

#### Create an HTTP Handler

```go
svc := NewService("example-service")
_ = svc.Register(exampleOperation, &myArbitraryLengthOperation{})
reg := NewServiceRegistry()
_ = reg.Register(svc)
handler, _ = reg.NewHandler()

httpHandler := nexus.NewHTTPHandler(nexus.HandlerOptions{
	Handler: handler,
})

listener, _ := net.Listen("tcp", "localhost:0")
// Handler URLs can be prefixed by using a request multiplexer (e.g. https://pkg.go.dev/net/http#ServeMux).
_ = http.Serve(listener, httpHandler)
```

#### Respond Synchronously with Failure

```go
func (h *myArbitraryLengthOperation) Start(ctx context.Context, input MyInput, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[MyOutput], error) {
	return nil, &nexus.UnsuccessfulOperationError{
		State: nexus.OperationStateFailed, // or OperationStateCanceled
		Failure: &nexus.Failure{Message: "Do or do not, there is not try"},
	}
}
```

#### Get Operation Result

The `GetResult` method is used to deliver an operation's result inline. If this method does not return an error, the
operation is considered as successfully completed. Return an `UnsuccessfulOperationError` to indicate completion or an
`ErrOperationStillRunning` error to indicate that the operation is still running.

When `GetOperationResultOptions.Wait` is greater than zero, this request should be treated as a long poll. Long poll
requests have a server side timeout, configurable via `HandlerOptions.GetResultTimeout`, and exposed via context
deadline. The context deadline is decoupled from the application level Wait duration.

It is the implementor's responsiblity to respect the client's wait duration and return in a timely fashion.
Consider using a derived context that enforces the wait timeout when implementing this method and return
`ErrOperationStillRunning` when that context expires as shown in the example.

```go
func (h *myArbitraryLengthOperation) GetResult(ctx context.Context, id string, options nexus.GetOperationResultOptions) (MyOutput, error) {
	if options.Wait > 0 { // request is a long poll
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Wait)
		defer cancel()

		result, err := h.pollOperation(ctx, options.Wait)
		if err != nil {
			// Translate deadline exceeded to "OperationStillRunning", this may or may not be semantically correct for
			// your application.
			// Some applications may want to "peek" the current status instead of performing this blind conversion if
			// the wait time is exceeded and the request's context deadline has not yet exceeded.
			if ctx.Err() != nil {
				return nil, nexus.ErrOperationStillRunning
			}
			// Optionally translate to operation failure (could also result in canceled state).
			// Optionally expose the error details to the caller.
			return nil, &nexus.UnsuccessfulOperationError{State: nexus.OperationStateFailed, Failure: nexus.Failure{Message: err.Error()}}
		}
		return result, nil
	} else {
		result, err := h.peekOperation(ctx)
		if err != nil {
			// Optionally translate to operation failure (could also result in canceled state).
			return nil, &nexus.UnsuccessfulOperationError{State: nexus.OperationStateFailed, Failure: nexus.Failure{Message: err.Error()}}
		}
		return result, nil
	}
}
```

#### Handle Asynchronous Completion

Implement `CompletionHandler.CompleteOperation` to get async operation completions.

```go
type myCompletionHandler struct {}

httpHandler := nexus.NewCompletionHTTPHandler(nexus.CompletionHandlerOptions{
	Handler: &myCompletionHandler{},
})

func (h *myCompletionHandler) CompleteOperation(ctx context.Context, completion *nexus.CompletionRequest) error {
	switch completion.State {
	case nexus.OperationStateCanceled, case nexus.OperationStateFailed:
		// completion.Failure will be popluated here
	case nexus.OperationStateSucceeded:
		// read completion.HTTPRequest Header and Body
	}
	return nil
}
```

#### Fail a Request

Returning an arbitrary error from any of the `Operation` and `CompletionHandler` methods will result in the error being
logged and the request responded to with a generic Internal Server Error status code and Failure message.

To fail a request with a custom status code and failure message, return a `nexus.HandlerError` as the error.
The error can either be constructed directly or with the `HandlerErrorf` helper.

```go
func (h *myArbitraryLengthOperation) Start(ctx context.Context, input MyInput, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[MyOutput], error) {
	return nil, nexus.HandlerErrorf(nexus.HandlerErrorTypeBadRequest, "unmet expectation")
}
```

### Logging

The handlers log internally and accept a `log/slog.Logger` to customize their log output, defaults to `slog.Default()`.

## Failure Structs

`nexus` exports a `Failure` struct that is used in both the client and handlers to represent both application level
operation failures and framework level HTTP request errors.

`Failure`s typically contain a single `Message` string but may also convey arbitrary JSONable `Details` and `Metadata`.

The `Details` field is encoded and it is up to the library user to encode to and decode from it.

## Contributing

### Prerequisites

- [Go 1.21](https://go.dev/doc/install)
- [golangci-lint](https://golangci-lint.run/usage/install/)

### Test

```shell
go test -v ./...
```

### Lint

```shell
golangci-lint run --verbose --timeout 1m --fix=false
```
