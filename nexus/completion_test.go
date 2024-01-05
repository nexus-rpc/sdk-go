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
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid URL path: %q", completion.HTTPRequest.URL.Path)
	}
	if completion.HTTPRequest.URL.Query().Get("a") != "b" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'a' query param: %q", completion.HTTPRequest.URL.Query().Get("a"))
	}
	if completion.HTTPRequest.Header.Get("foo") != "bar" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'foo' header: %q", completion.HTTPRequest.Header.Get("foo"))
	}
	if completion.HTTPRequest.Header.Get("User-Agent") != userAgent {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'User-Agent' header: %q", completion.HTTPRequest.Header.Get("User-Agent"))
	}
	var result int
	err := completion.Result.Consume(&result)
	if err != nil {
		return err
	}
	if result != 666 {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid result: %q", result)
	}
	return nil
}

func TestSuccessfulCompletion(t *testing.T) {
	ctx, callbackURL, teardown := setupForCompletion(t, &successfulCompletionHandler{}, nil)
	defer teardown()

	completion, err := NewOperationCompletionSuccessful(666, OperationCompletionSuccesfulOptions{})
	completion.Header.Add("foo", "bar")
	require.NoError(t, err)

	request, err := NewCompletionHTTPRequest(ctx, callbackURL, completion)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
}

func TestSuccessfulCompletion_CustomSerializr(t *testing.T) {
	serializer := &customSerializer{}
	ctx, callbackURL, teardown := setupForCompletion(t, &successfulCompletionHandler{}, serializer)
	defer teardown()

	completion, err := NewOperationCompletionSuccessful(666, OperationCompletionSuccesfulOptions{
		Serializer: serializer,
	})
	completion.Header.Add("foo", "bar")
	require.NoError(t, err)

	request, err := NewCompletionHTTPRequest(ctx, callbackURL, completion)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)

	require.Equal(t, 1, serializer.decoded)
	require.Equal(t, 1, serializer.encoded)
}

type failureExpectingCompletionHandler struct {
}

func (h *failureExpectingCompletionHandler) CompleteOperation(ctx context.Context, completion *CompletionRequest) error {
	if completion.State != OperationStateCanceled {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "unexpected completion state: %q", completion.State)
	}
	if completion.Failure.Message != "expected message" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid failure: %v", completion.Failure)
	}
	if completion.HTTPRequest.Header.Get("foo") != "bar" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'foo' header: %q", completion.HTTPRequest.Header.Get("foo"))
	}

	return nil
}

func TestFailureCompletion(t *testing.T) {
	ctx, callbackURL, teardown := setupForCompletion(t, &failureExpectingCompletionHandler{}, nil)
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
	return HandlerErrorf(HandlerErrorTypeBadRequest, "I can't get no satisfaction")
}

func TestBadRequestCompletion(t *testing.T) {
	ctx, callbackURL, teardown := setupForCompletion(t, &failingCompletionHandler{}, nil)
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
