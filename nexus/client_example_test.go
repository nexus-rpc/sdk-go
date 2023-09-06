package nexus_test

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/nexus-rpc/sdk-go/nexus"
)

type MyStruct struct {
	Field string
}

var ctx = context.Background()
var client *nexus.Client

func ExampleClient_StartOperation() {
	options, _ := nexus.NewStartOperationOptions("example", MyStruct{Field: "value"})
	result, err := client.StartOperation(ctx, options)
	if err != nil {
		var unsuccessfulOperationError *nexus.UnsuccessfulOperationError
		if errors.As(err, &unsuccessfulOperationError) { // operation failed or canceled
			fmt.Printf("Operation unsuccessful with state: %s, failure message: %s\n", unsuccessfulOperationError.State, unsuccessfulOperationError.Failure.Message)
		}
		// handle error here
	}
	if result.Successful != nil { // operation successful
		response := result.Successful
		// must close the returned response body and read it until EOF to free up the underlying connection
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		fmt.Printf("Got response with content type: %s, body first bytes: %v\n", response.Header.Get("Content-Type"), body[:5])
	} else { // operation started asynchronously
		handle := result.Pending
		fmt.Printf("Started asynchronous operation with ID: %s\n", handle.ID)
	}
}

func ExampleClient_ExecuteOperation() {
	options, _ := nexus.NewExecuteOperationOptions("operation name", MyStruct{Field: "value"})
	response, err := client.ExecuteOperation(ctx, options)
	if err != nil {
		// handle nexus.UnsuccessfulOperationError, nexus.ErrOperationStillRunning and, context.DeadlineExceeded
	}
	// must close the returned response body and read it until EOF to free up the underlying connection
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	fmt.Printf("Got response with content type: %s, body first bytes: %v\n", response.Header.Get("Content-Type"), body[:5])
}
