package nexus_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/nexus-rpc/sdk-go/nexus"
)

type MyStruct struct {
	Field string
}

var ctx = context.Background()
var client *nexus.ServiceClient

func ExampleServiceClient_StartOperation() {
	response, err := client.StartOperation(ctx, "example", MyStruct{Field: "value"}, nexus.StartOperationOptions{})
	if err != nil {
		var handlerError *nexus.HandlerError
		if errors.As(err, &handlerError) {
			fmt.Printf("Handler returned an error, type: %s, failure message: %s\n", handlerError.Type, handlerError.Cause.Error())
		}
		// most other errors should be returned as *nexus.TransportError
	}
	if response.Sync() != nil { // operation complete
		result, opErr := response.Sync().Get()
		if opErr != nil {
			var operationErr *nexus.OperationError
			if errors.As(err, &operationErr) { // operation failed or canceled
				fmt.Printf("Operation unsuccessful with state: %s, failure message: %s\n", operationErr.State, operationErr.Cause.Error())
			} else {
				fmt.Printf("Operation unsuccessful with unexpected error: %v", opErr)
			}
		} else { // operation successful
			// must consume the result to free up the underlying connection
			var output MyStruct
			_ = result.Consume(&output)
			fmt.Printf("Got response: %v\n", output)
		}
	} else { // operation started asynchronously
		handle := response.Async()
		fmt.Printf("Started asynchronous operation with token: %s\n", handle.Token)
	}
}

func ExampleServiceClient_ExecuteOperation() {
	response, err := client.ExecuteOperation(ctx, "operation name", MyStruct{Field: "value"}, nexus.ExecuteOperationOptions{})
	if err != nil {
		// handle nexus.OperationError, nexus.ErrOperationStillRunning and, context.DeadlineExceeded
	}
	// must close the returned response body and read it until EOF to free up the underlying connection
	var output MyStruct
	_ = response.Consume(&output)
	fmt.Printf("Got response: %v\n", output)
}
