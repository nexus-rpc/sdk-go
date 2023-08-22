package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexusapi"
	"github.com/nexus-rpc/sdk-go/nexusclient"
	"github.com/nexus-rpc/sdk-go/nexusserver"
	"github.com/stretchr/testify/require"
)

type successfulCompletionHandler struct {
}

func (h *successfulCompletionHandler) Complete(ctx context.Context, completion *nexusserver.CompletionRequest) error {
	if completion.HTTPRequest.URL.Path != "/callback" {
		return newBadRequestError("invalid URL path: %q", completion.HTTPRequest.URL.Path)
	}
	if completion.HTTPRequest.URL.Query().Get("a") != "b" {
		return newBadRequestError("invalid 'a' query param: %q", completion.HTTPRequest.URL.Query().Get("a"))
	}
	return nil
}

func TestSuccessfulCompletion(t *testing.T) {
	ctx, client, callbackURL, teardown := setupForCompletion(t, &successfulCompletionHandler{})
	defer teardown()

	err := client.DeliverCompletion(ctx, callbackURL, nexusclient.NewBytesSuccessfulOperationCompletion(nil, []byte("success")))
	require.NoError(t, err)
}

type failureExpectingCompletionHandler struct {
}

func (h *failureExpectingCompletionHandler) Complete(ctx context.Context, completion *nexusserver.CompletionRequest) error {
	if completion.State != nexusapi.OperationStateCanceled {
		return newBadRequestError("unexpected completion state: %q", completion.State)
	}
	if completion.Failure.Message != "expected message" {
		return newBadRequestError("invalid failure: %v", completion.Failure)
	}
	return nil
}

func TestFailureCompletion(t *testing.T) {
	ctx, client, callbackURL, teardown := setupForCompletion(t, &failureExpectingCompletionHandler{})
	defer teardown()

	err := client.DeliverCompletion(ctx, callbackURL, &nexusclient.UnsuccessfulOperationCompletion{
		State: nexusapi.OperationStateCanceled,
		Failure: &nexusapi.Failure{
			Message: "expected message",
		},
	})
	require.NoError(t, err)
}

type failingCompletionHandler struct {
}

func (h *failingCompletionHandler) Complete(ctx context.Context, completion *nexusserver.CompletionRequest) error {
	return newBadRequestError("I can't get no satisfaction")
}

func TestBadRequestCompletion(t *testing.T) {
	ctx, client, callbackURL, teardown := setupForCompletion(t, &failingCompletionHandler{})
	defer teardown()

	err := client.DeliverCompletion(ctx, callbackURL, nexusclient.NewBytesSuccessfulOperationCompletion(nil, []byte("success")))
	var unexpectedResponseError *nexusclient.UnexpectedResponseError
	require.ErrorAs(t, err, &unexpectedResponseError)
	require.Equal(t, http.StatusBadRequest, unexpectedResponseError.Response.StatusCode)
}
