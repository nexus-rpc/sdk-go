# Nexus Go SDK

Client and server libraries for working with the Nexus [HTTP API](https://github.com/nexus-rpc/api).

## Installation

### Intall the Client

```shell
go get github.com/nexus-rpc/nexusclient
```

### Intall the Server

```
go get github.com/nexus-rpc/nexusserver
```

## Usage

### Client

The Nexus Client is used to start operations, get [handles](#operationhandle) to existing operations, and deliver operation
completions.

#### Create a Client

```go
use (
	"github.com/nexus-rpc/sdk-go/nexusclient"
)

client, err := nexusclient.NewClient(nexusclient.Options{
	ServiceBaseURL: "https://example.com/path/to/my/service",
})
```

#### Start an Operation

```go
// request is a StartOperationRequest that can also be constructed directly.
// See the StartOperationRequest definition for advanced options, such as setting a request ID, and arbitrary HTTP
// headers.
request, _ := nexusclient.NewJSONStartOperationRequest("operation name", MyStruct{Field: "value"})

handle, _ := client.StartOperation(ctx, request)
// Handles may hold references to an HTTP response body and must be explicitly closed.
defer handle.Close()
```

#### Get a Handle to an Existing Operation

Getting a handle does not incur a trip to the server.

```go
handle := client.GetHandle("operation name", "operation ID")
defer handle.Close()
```

### OperationHandle

`OperationHandle`s are used to cancel and get the result and status of an operation.

Handles expose a few getters: `Operation()`, `ID()` and `State()`.

#### Operation

`Operation` is the name of the operation this handle represents.

#### ID

`ID` is the operation ID as returned by a Nexus handler in the response to `StartOperation` or set by the client in the
`GetHandle` method. ID may be empty in case the handle represents an operation that completed synchronously.

#### State

`State()` the last known operation state. Empty for handles created with `client.GetHandle` before issuing a request to
get the result.

#### Get the Result of an Operation

The `GetResult` method is used to get the result of an operation.

If the handle was obtained using the `StartOperation` API, and the operation completed synchronously, the response may
already be stored in the handle, otherwise, the handle will use its associated client to issue a request to get the
operation's result.

By default, `GetResult` returns a `nil` response immediately and no error after issuing a call if the operation has not
yet completed.

Callers may set `GetResultOptions.Wait` to true to alter this behavior, causing the client to long poll for the result
until the provided context deadline is exceeded. When the deadline exceeds, `GetResult` will return a `nil` response and
`context.DeadlineExceeded` error. The client may issue multiple requests until the deadline exceeds with a max request
timeout of `Options.GetResultMaxRequestTimeout`.

Custom HTTP headers may be provided via `GetResultOptions`.

```go
response, err := handle.GetResult(ctx, nexusclient.GetResultOptions{})
// response type is an *http.Response
```

When an operation completes unsuccessfuly, the returned error type is `UnsuccessfulOperationError`.

```go
_, err := handle.GetResult(ctx, nexusclient.GetResultOptions{})
var unsuccessfulOperationError *UnsuccessfulOperationError
if errors.As(err, &unsuccessfulOperationError) {
	fmt.Println(
		"State:",
		unsuccessfulOperationError.State,
		"failure message:",
		unsuccessfulOperationError.Failure.Message,
	)
}
```

#### Get Operation Information

The `GetInfo` method is used to get operation information (currently only the operation's state) issuing a network
request to the service handler.

⚠️ Getting info does **not** update the handle's `State`.

Custom HTTP headers may be provided via `GetInfoOptions`.

```go
info, _ := handle.GetInfo(ctx, nexusclient.GetInfoOptions{})
```

#### Cancel an Operation

The `Cancel` method requests cancelation of an asynchronous operation.

Cancelation in Nexus is asynchronous and may be not be respected by the operation's implementation.

Custom HTTP headers may be provided via `CancelOptions`.

```go
_ := handle.Cancel(ctx, nexusclient.CancelOptions{})
```

#### Complete an Operation

Handlers starting asynchronous operations may need to deliver responses via a caller specified callback URL.

`DeliverCompletion` is used to deliver operation completions - successful or unsuccessful - to the provided callback
URL.

To deliver successful completions, pass a `OperationCompletionSuccessful` struct pointer, which may also be constructed
with the `NewJSONOperationCompletionSuccessful` helper.

Custom HTTP headers may be provided via `OperationCompletionSuccessful.Header`.

```go
completion, _ := NewOperationCompletionSuccessfulJSON(MyStruct{Field: "value"})
_ = client.DeliverCompletion(ctx, completion)
```

To deliver failed and canceled completions, pass a `OperationCompletionUnsuccessful` struct pointer with the failure and
state attached.

Custom HTTP headers may be provided via `OperationCompletionUnsuccessful.Header`.

```go
completion := &OperationCompletionUnsuccessful{
	State: nexusapi.OperationStateCanceled,
	Failure: &nexusapi.Failure{Message: "canceled as requested"},
}
_ = client.DeliverCompletion(ctx, completion)
```

### Server

The `nexusserver` package provides a user friendly API for handling Nexus HTTP requests.

It exports a couple of user implementable interfaces: `Handler` and `CompletionHandler`.

- `Handler` exposes the entire Nexus operation API for starting, canceling, getting result and information of
  operations.
- `CompletionHandler` exposes an API to handle async delivery of operation completions.

#### Create an HTTP Handler

```go
type myHandler struct {}

// Implement handler methods here ...

httpHandler := nexusserver.NewHTTPHandler(nexusserver.Options{
	Handler: &myHandler,
})

listener, err := net.Listen("tcp", "localhost:0")
require.NoError(t, err)
// Handler URLs can be prefixed by using a router.
_ = http.Serve(listener, httpHandler)
```

#### Start an Operation

##### Respond Synchronously

Return an `OperationResponseSync` from `StartOperation`, delivering the operation result.

Use the `NewJSONOperationResponseSync` and `NewBytesOperationResponseSync` helpers for simple responses.

`StartOperationRequest` contains the original `http.Request` for extraction of headers, URL, and request body.

Custom response headers may be provided via `OperationResponseSync.Header`.

```go
func (h *myHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return nexusserver.NewJSONOperationResponseSync(MyStruct{Field: "value"}), nil
}
```

##### Indicate that an Operation Completes Asynchronously

```go
func (h *myHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return &nexusserver.OperationResponseAsync{OperationID: "async"}, nil
}
```

##### Respond Synchronously with Failure

```go
func (h *myHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return nil, &nexusapi.UnsuccessfulOperationError{
		State: nexusapi.OperationStateFailed, // or OperationStateCanceled
		Failure: &nexusapi.Failure{Message: "Do or do not, there is not try"},
	}
}
```

#### Cancel an Operation

`CancelOperationRequest` contains the original `http.Request` for extraction of headers, URL, and other useful
information.

```go
func (h *myHandler) CancelOperation(ctx context.Context, request *nexusserver.CancelOperationRequest) error {
	fmt.Println("Canceling", request.Operation, "with ID:", request.OperationID)
	return nil
}
```

#### Get Operation Info

`GetOperationInfoRequest` contains the original `http.Request` for extraction of headers, URL, and other useful
information.

```go
func (h *myHandler) GetOperationInfo(ctx context.Context, request *nexusserver.GetOperationInfoRequest) (*nexusapi.OperationInfo, error) {
	fmt.Println("Getting info for", request.Operation)
	return &nexusapi.OperationInfo{
		ID:    request.OperationID,
		State: nexusapi.OperationStateRunning,
	}, nil
}
```

#### Get Operation Result

The `GetOperationResult` method is used to deliver an operation's result inline. Similarly to `StartOperation`, this
method should return an `OperationResponseSync` or fail with an `UnsuccessfulOperationError` to indicate completion or
an `OperationResponseAsync` to indicate that the operation is still running.
The method may also return a `context.DeadlineExceeded` error to indicate that the operation is still running.

`GetOperationResultRequest.Wait` indicates whether the caller is willing to wait for the operation to complete. When
set, context deadline indicates how long the caller is willing to wait for, capped by
`Options.GetResultMaxRequestTimeout`.

`GetOperationResultRequest` contains the original `http.Request` for extraction of headers, URL, and other useful
information.

```go
func (h *myHandler) GetOperationResult(ctx context.Context, request *nexusserver.GetOperationResultRequest) (nexusserver.OperationResponse, error) {
	someResult, err := getResultOfMyOperation(ctx, request.Operation, request.OperationID)
	if err != nil {
		return nil, err
	}
	return nexusserver.NewJSONOperationResponseSync(someResult), nil
}
```

#### Handle Asynchronous Completion

Implement `CompletionHandler.CompleteOperation` to get async operation completions.

```go
httpHandler := nexusserver.NewCompletionHTTPHandler(nexusserver.CompletionOptions{
	Handler: &myCompletionHandler{},
})

func (h *myCompletionHandler) CompleteOperation(ctx context.Context, completion *nexusserver.CompletionRequest) error {
	switch completion.State {
	case nexusapi.OperationStateCanceled, case nexusapi.OperationStateFailed:
		// completion.Failure will be popluated here
	case nexusapi.OperationStateSucceeded:
		// read completion.HTTPRequest Header and Body
	}
	return nil
}
```

### Fail a Request

Returning an error from any of the `Handler` and `CompletionHandler` methods will result in the error being logged and
the request responded with a generic Internal Server Error status code and Failure message.

To fail a request with a custom status code and failure message, return a `nexusserver.HandlerError` as the error.

```go
func (h *myHandler) StartOperation(ctx context.Context, request *nexusserver.StartOperationRequest) (nexusserver.OperationResponse, error) {
	return nil, &nexusserver.HandlerError{
		StatusCode: http.StatusBadRequest,
		Failure: &nexusapi.Failure{Message: "unmet expectation"},
	}
}
```

### Logging

Both the client and server packages log internally and accept a `log/slog.Handler` to customize their log output.

By default logs are emited in textual format to stderr at INFO level.

## Failure Structs

`nexusapi` exports a `Failure` struct that is used in both the client and server packages to represent both application
level operation failures and framework level HTTP request errors.

`Failure`s typically contain a single `Message` string but may also convey arbitrary JSONable `Details` and `Metadata`.

The `Details` field is encoded and it is up to the library user to encode to and decode from it.

## Contributing

### Prerequisites

- [Go 1.21](https://go.dev/doc/install)
- [golangci-lint](https://golangci-lint.run/usage/install/)

### Test

#### On Unix

```shell
go test -v  $(go list -f '{{.Dir}}/...' -m | xargs)
```

#### On Windows

> TODO

### Lint

#### On Unix

```shell
golangci-lint --verbose --timeout 10m --fix=false $(go list -f '{{.Dir}}/...' -m | xargs)
```

#### On Windows

> TODO
