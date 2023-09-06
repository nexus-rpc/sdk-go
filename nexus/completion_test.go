package nexus

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type successfulCompletionHandler struct {
}

func (h *successfulCompletionHandler) CompleteOperation(ctx context.Context, completion *CompletionRequest) error {
	if completion.HTTPRequest.URL.Path != "/callback" {
		return newBadRequestError("invalid URL path: %q", completion.HTTPRequest.URL.Path)
	}
	if completion.HTTPRequest.URL.Query().Get("a") != "b" {
		return newBadRequestError("invalid 'a' query param: %q", completion.HTTPRequest.URL.Query().Get("a"))
	}
	if completion.HTTPRequest.Header.Get("foo") != "bar" {
		return newBadRequestError("invalid 'foo' header: %q", completion.HTTPRequest.Header.Get("foo"))
	}
	if completion.HTTPRequest.Header.Get("User-Agent") != userAgent {
		return newBadRequestError("invalid 'User-Agent' header: %q", completion.HTTPRequest.Header.Get("User-Agent"))
	}
	b, err := io.ReadAll(completion.HTTPRequest.Body)
	if err != nil {
		return err
	}
	if !bytes.Equal(b, []byte("success")) {
		return newBadRequestError("invalid request body: %q", b)
	}
	return nil
}

func TestSuccessfulCompletion(t *testing.T) {
	ctx, callbackURL, teardown := setupForCompletion(t, &successfulCompletionHandler{})
	defer teardown()

	request, err := NewCompletionHTTPRequest(ctx, callbackURL, &OperationCompletionSuccessful{
		Header: http.Header{"foo": []string{"bar"}},
		Body:   bytes.NewReader([]byte("success")),
	})
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
}

type failureExpectingCompletionHandler struct {
}

func (h *failureExpectingCompletionHandler) CompleteOperation(ctx context.Context, completion *CompletionRequest) error {
	if completion.State != OperationStateCanceled {
		return newBadRequestError("unexpected completion state: %q", completion.State)
	}
	if completion.Failure.Message != "expected message" {
		return newBadRequestError("invalid failure: %v", completion.Failure)
	}
	if completion.HTTPRequest.Header.Get("foo") != "bar" {
		return newBadRequestError("invalid 'foo' header: %q", completion.HTTPRequest.Header.Get("foo"))
	}

	return nil
}

func TestFailureCompletion(t *testing.T) {
	ctx, callbackURL, teardown := setupForCompletion(t, &failureExpectingCompletionHandler{})
	defer teardown()

	request, err := NewCompletionHTTPRequest(ctx, callbackURL, &OperationCompletionUnsuccessful{
		Header: http.Header{"foo": []string{"bar"}},
		State:  OperationStateCanceled,
		Failure: &Failure{
			Message: "expected message",
		},
	})
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
}

type failingCompletionHandler struct {
}

func (h *failingCompletionHandler) CompleteOperation(ctx context.Context, completion *CompletionRequest) error {
	return newBadRequestError("I can't get no satisfaction")
}

func TestBadRequestCompletion(t *testing.T) {
	ctx, callbackURL, teardown := setupForCompletion(t, &failingCompletionHandler{})
	defer teardown()

	request, err := NewCompletionHTTPRequest(ctx, callbackURL, &OperationCompletionSuccessful{Body: bytes.NewReader([]byte("success"))})
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}
