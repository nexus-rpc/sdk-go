package nexus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceBaseURL(t *testing.T) {
	var err error

	_, err = NewClient(ClientOptions{ServiceBaseURL: ""})
	require.ErrorIs(t, err, errEmptyServiceBaseURL)

	_, err = NewClient(ClientOptions{ServiceBaseURL: "-http://invalid"})
	var urlError *url.Error
	require.ErrorAs(t, err, &urlError)

	_, err = NewClient(ClientOptions{ServiceBaseURL: "smtp://example.com"})
	require.ErrorIs(t, err, errInvalidURLScheme)

	_, err = NewClient(ClientOptions{ServiceBaseURL: "http://example.com"})
	require.NoError(t, err)

	_, err = NewClient(ClientOptions{ServiceBaseURL: "https://example.com"})
	require.NoError(t, err)
}

func TestGetResultMaxRequestTimeout(t *testing.T) {
	var err error
	var client *Client

	client, err = NewClient(ClientOptions{
		ServiceBaseURL:      "http://unit.test",
		GetResultMaxTimeout: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, time.Second, client.options.GetResultMaxTimeout)

	// Default is set
	client, err = NewClient(ClientOptions{
		ServiceBaseURL: "http://unit.test",
	})
	require.NoError(t, err)
	require.Equal(t, time.Minute, client.options.GetResultMaxTimeout)
}

var client Client
var ctx context.Context

type MyStruct struct {
	Field string
}

func ExampleClient_StartOperation() {
	options, _ := NewStartOperationOptions("example", MyStruct{Field: "value"})
	result, err := client.StartOperation(ctx, options)
	if err != nil {
		var unsuccessfulOperationError *UnsuccessfulOperationError
		if errors.As(err, &unsuccessfulOperationError) { // operation failed or canceled
			fmt.Printf("Operation unsuccessful with state: %s, failure message: %s\n", unsuccessfulOperationError.State, unsuccessfulOperationError.Failure.Message)
		}
		// Handle error here
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
}
