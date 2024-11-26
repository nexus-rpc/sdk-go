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
var client *nexus.HTTPClient

func ExampleClient_StartOperation() {
	result, err := client.StartOperation(ctx, "example", MyStruct{Field: "value"}, nexus.StartOperationOptions{})
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
		response := result.Successful
		// must consume the response to free up the underlying connection
		var output MyStruct
		_ = response.Consume(&output)
		fmt.Printf("Got response: %v\n", output)
	} else { // operation started asynchronously
		handle := result.Pending
		fmt.Printf("Started asynchronous operation with ID: %s\n", handle.ID)
	}
}

func ExampleClient_ExecuteOperation() {
	response, err := client.ExecuteOperation(ctx, "operation name", MyStruct{Field: "value"}, nexus.ExecuteOperationOptions{})
	if err != nil {
		// handle nexus.UnsuccessfulOperationError, nexus.ErrOperationStillRunning and, context.DeadlineExceeded
	}
	// must close the returned response body and read it until EOF to free up the underlying connection
	var output MyStruct
	_ = response.Consume(&output)
	fmt.Printf("Got response: %v\n", output)
}
