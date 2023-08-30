# Nexus Go SDK

Client and server package for working with the Nexus [HTTP API](https://github.com/nexus-rpc/api).

**⚠️ EXPERIMENTAL ⚠️**

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

The Nexus Client is used to start operations, get [handles](#operationhandle) to existing operations, and deliver operation
completions.

#### Create a Client

```go
client, err := nexus.NewClient(nexus.ClientOptions{
	ServiceBaseURL: "https://example.com/path/to/my/service",
})
```

#### Start an Operation

```go
// request is a StartOperationRequest that can also be constructed directly.
// See the StartOperationRequest definition for advanced options, such as setting a request ID, and arbitrary HTTP
// headers.
options, _ := NewStartOperationOptions("example", MyStruct{Field: "value"})
result, err := client.StartOperation(ctx, options)
if err != nil {
	var unsuccessfulOperationError *nexus.UnsuccessfulOperationError
	if errors.As(err, &unsuccessfulOperationError) { // operation failed or canceled
		fmt.Printf("Operation unsuccessful with state: %s, failure message: %s\n", unsuccessfulOperationError.State, unsuccessfulOperationError.Failure.Message)
	}
	return err
}
if result.Successful != nil { // operation successful
	response := result.Successful
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	fmt.Printf("Got response with content type: %s, body first bytes: %v\n", response.Header.Get("Content-Type"), body[:5])
} else { // operation started asynchronously
	handle := result.Pending
	fmt.Printf("Started asynchronous operation with ID: %s\n", handle.ID)
}
```

#### Start an Operation and Await its Completion

The Client provides the `ExecuteOperation` helper function as a shorthand for `StartOperation` and issuing a `GetResult`
in case the operation is asynchronous.

```go
options, _ := nexus.NewExecuteOperationOptions("operation name", MyStruct{Field: "value"})
response, err := client.ExecuteOperation(ctx, options)
if err != nil {
	// handle similarly to StartOperation errors above
}
// must close the returned response body and read it until EOF to free up the underlying connection
defer response.Body.Close()
fmt.Printf("Got response with content type: %s\n", response.Header.Get("Content-Type"))
body, _ := io.ReadAll(response.Body)
```

#### Get a Handle to an Existing Operation

Getting a handle does not incur a trip to the server.

```go
handle, err := client.NewHandle("operation name", "operation ID")
```

### OperationHandle

`OperationHandle`s are used to cancel and get the result and status of an operation.

Handles expose a couple of readonly attributes: `Operation`, `ID`.

#### Operation

`Operation` is the name of the operation this handle represents.

#### ID

`ID` is the operation ID as returned by a Nexus handler in the response to `StartOperation` or set by the client in the
`NewHandle` method.

#### Get the Result of an Operation

The `GetResult` method is used to get the result of an operation, issuing a network request to the handle's client's
configured endpoint.

By default, GetResult returns (nil, [ErrOperationStillRunning]) immediately after issuing a call if the operation has
not yet completed.

Callers may set GetOperationResultOptions.Wait to a value greater than 0 to alter this behavior, causing the client to
long poll for the result issuing one or more requests until the provided wait period exceeds, in which case (nil,
[ErrOperationStillRunning]) is returned.

The wait time is capped to the deadline of the provided context.

When an operation completes unsuccessfuly, the returned error type is `UnsuccessfulOperationError`.

⚠️ If a response is returned, its body must be read in its entirety and closed to free up the underlying connection.

Custom HTTP headers may be provided via `GetOperationResultOptions`.

```go
response, err := handle.GetResult(ctx, nexus.GetOperationResultOptions{})
defer response.Body.Close()
if err != nil {
	// handle similarly to StartOperation errors above
}
// response type is an *http.Response
```

#### Get Operation Information

The `GetInfo` method is used to get operation information (currently only the operation's state) issuing a network
request to the service handler.

Custom HTTP headers may be provided via `GetOperationInfoOptions`.

```go
info, _ := handle.GetInfo(ctx, nexus.GetOperationInfoOptions{})
```

#### Cancel an Operation

The `Cancel` method requests cancelation of an asynchronous operation.

Cancelation in Nexus is asynchronous and may be not be respected by the operation's implementation.

Custom HTTP headers may be provided via `CancelOperationOptions`.

```go
_ := handle.Cancel(ctx, nexus.CancelOperationOptions{})
```

#### Complete an Operation

Handlers starting asynchronous operations may need to deliver responses via a caller specified callback URL.

`DeliverCompletion` is used to deliver operation completions - successful or unsuccessful - to the provided callback
URL.

To deliver successful completions, pass a `OperationCompletionSuccessful` struct pointer, which may also be constructed
with the `NewOperationCompletionSuccessful` helper.

Custom HTTP headers may be provided via `OperationCompletionSuccessful.Header`.

```go
client, _ := nexus.NewCompletionClient(nexus.CompletionClientOptions{})
completion, _ := nexus.NewOperationCompletionSuccessful(MyStruct{Field: "value"})
_ = client.DeliverCompletion(ctx, completion)
```

To deliver failed and canceled completions, pass a `OperationCompletionUnsuccessful` struct pointer with the failure and
state attached.

Custom HTTP headers may be provided via `OperationCompletionUnsuccessful.Header`.

```go
completion := &OperationCompletionUnsuccessful{
	State: nexus.OperationStateCanceled,
	Failure: &nexus.Failure{Message: "canceled as requested"},
}
_ = client.DeliverCompletion(ctx, completion)
```

### Server

The nexus package exposes a couple of user implementable interfaces for handling API requests: `Handler` and
`CompletionHandler`.

- `Handler` exposes the entire Nexus operation API for starting, canceling, getting result and information of
  operations.
- `CompletionHandler` exposes an API to handle async delivery of operation completions.

#### Create an HTTP Handler

```go
type myHandler struct {
	nexus.UnimplementedHandler
}

// Implement handler methods here ...

httpHandler := nexus.NewHTTPHandler(nexus.HandlerOptions{
	Handler: &myHandler,
})

listener, _ := net.Listen("tcp", "localhost:0")
// Handler URLs can be prefixed by using a router.
_ = http.Serve(listener, httpHandler)
```

#### Start an Operation

##### Respond Synchronously

Return an `OperationResponseSync` from `StartOperation`, delivering the operation result.

Use the `NewOperationResponseSync` helper for JSON responses.

`StartOperationRequest` contains the original `http.Request` for extraction of headers, URL, and request body.

Custom response headers may be provided via `OperationResponseSync.Header`.

```go
func (h *myHandler) StartOperation(ctx context.Context, request *nexus.StartOperationRequest) (nexus.OperationResponse, error) {
	return nexus.NewOperationResponseSync(MyStruct{Field: "value"}), nil
}
```

##### Indicate that an Operation Completes Asynchronously

```go
func (h *myHandler) StartOperation(ctx context.Context, request *nexus.StartOperationRequest) (nexus.OperationResponse, error) {
	return &nexus.OperationResponseAsync{OperationID: "async"}, nil
}
```

##### Respond Synchronously with Failure

```go
func (h *myHandler) StartOperation(ctx context.Context, request *nexus.StartOperationRequest) (nexus.OperationResponse, error) {
	return nil, &nexus.UnsuccessfulOperationError{
		State: nexus.OperationStateFailed, // or OperationStateCanceled
		Failure: &nexus.Failure{Message: "Do or do not, there is not try"},
	}
}
```

#### Cancel an Operation

`CancelOperationRequest` contains the original `http.Request` for extraction of headers, URL, and other useful
information.

```go
func (h *myHandler) CancelOperation(ctx context.Context, request *nexus.CancelOperationRequest) error {
	fmt.Println("Canceling", request.Operation, "with ID:", request.OperationID)
	return nil
}
```

#### Get Operation Info

`GetOperationInfoRequest` contains the original `http.Request` for extraction of headers, URL, and other useful
information.

```go
func (h *myHandler) GetOperationInfo(ctx context.Context, request *nexus.GetOperationInfoRequest) (*nexus.OperationInfo, error) {
	fmt.Println("Getting info for", request.Operation)
	return &nexus.OperationInfo{
		ID:    request.OperationID,
		State: nexus.OperationStateRunning,
	}, nil
}
```

#### Get Operation Result

The `GetOperationResult` method is used to deliver an operation's result inline. Similarly to `StartOperation`, this
method should return an `OperationResponseSync` or fail with an `UnsuccessfulOperationError` to indicate completion or
an `ErrOperationStillRunning` error to indicate that the operation is still running.
The method may also return a `context.DeadlineExceeded` error to indicate that the operation is still running.

`GetOperationResultRequest.Wait` indicates whether the caller is willing to wait for the operation to complete. When
set, context deadline indicates how long the caller is willing to wait for, capped by `HandlerOptions.GetResultMaxTimeout`.

`GetOperationResultRequest` contains the original `http.Request` for extraction of headers, URL, and other useful
information.

```go
func (h *myHandler) GetOperationResult(ctx context.Context, request *nexus.GetOperationResultRequest) (nexus.OperationResponse, error) {
	someResult, err := getResultOfMyOperation(ctx, request.Operation, request.OperationID)
	if err != nil {
		// returning context.DeadlineExceeded is equivalent to OperationResponseAsync
		return nil, err
	}
	return nexus.NewOperationResponseSync(someResult), nil
}
```

#### Handle Asynchronous Completion

Implement `CompletionHandler.CompleteOperation` to get async operation completions.

```go
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

### Fail a Request

Returning an error from any of the `Handler` and `CompletionHandler` methods will result in the error being logged and
the request responded to with a generic Internal Server Error status code and Failure message.

To fail a request with a custom status code and failure message, return a `nexus.HandlerError` as the error.

```go
func (h *myHandler) StartOperation(ctx context.Context, request *nexus.StartOperationRequest) (nexus.OperationResponse, error) {
	return nil, &nexus.HandlerError{
		StatusCode: http.StatusBadRequest,
		Failure: &nexus.Failure{Message: "unmet expectation"},
	}
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
