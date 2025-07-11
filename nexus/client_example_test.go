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
var client *nexus.Client

func ExampleClient_StartOperation() {
	response, err := client.StartOperation(ctx, "example", MyStruct{Field: "value"}, nexus.StartOperationOptions{})
	if err != nil {
		var handlerError *nexus.HandlerError
		if errors.As(err, &handlerError) {
			fmt.Printf("Handler returned an error, type: %s, failure message: %s\n", handlerError.Type, handlerError.Cause.Error())
		}
		// most other errors should be returned as *nexus.UnexpectedResponseError
	}
	if response.Complete != nil { // operation complete
		result, opErr := response.Complete.Get()
		if opErr != nil {
			var OperationError *nexus.OperationError
			if errors.As(err, &OperationError) { // operation failed or canceled
				fmt.Printf("Operation unsuccessful with state: %s, failure message: %s\n", OperationError.State, OperationError.Cause.Error())
			}
		} else { // operation successful
			// must consume the result to free up the underlying connection
			var output MyStruct
			_ = result.Consume(&output)
			fmt.Printf("Got response: %v\n", output)
		}
	} else { // operation started asynchronously
		handle := response.Pending
		fmt.Printf("Started asynchronous operation with token: %s\n", handle.Token)
	}
}

func ExampleClient_ExecuteOperation() {
	response, err := client.ExecuteOperation(ctx, "operation name", MyStruct{Field: "value"}, nexus.ExecuteOperationOptions{})
	if err != nil {
		// handle nexus.OperationError, nexus.ErrOperationStillRunning and, context.DeadlineExceeded
	}
	// must close the returned response body and read it until EOF to free up the underlying connection
	var output MyStruct
	_ = response.Consume(&output)
	fmt.Printf("Got response: %v\n", output)
}
