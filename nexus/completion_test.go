package nexus

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

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
	if completion.OperationID != "test-operation-id" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid %q header: %q", HeaderOperationID, completion.HTTPRequest.Header.Get(HeaderOperationID))
	}
	if len(completion.Links) == 0 {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "expected Links to be set on CompletionRequest")
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
	ctx, callbackURL, teardown := setupForCompletion(t, &successfulCompletionHandler{}, nil, nil)
	defer teardown()

	completion, err := NewOperationCompletionSuccessful(666, OperationCompletionSuccessfulOptions{
		OperationID: "test-operation-id",
		StartTime:   time.Now(),
		Links: []Link{{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/something",
				RawQuery: "param=value",
			},
			Type: "url",
		}},
	})
	completion.Header.Set("foo", "bar")
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

func TestSuccessfulCompletion_CustomSerializer(t *testing.T) {
	serializer := &customSerializer{}
	ctx, callbackURL, teardown := setupForCompletion(t, &successfulCompletionHandler{}, serializer, nil)
	defer teardown()

	completion, err := NewOperationCompletionSuccessful(666, OperationCompletionSuccessfulOptions{
		Serializer: serializer,
		Links: []Link{{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/something",
				RawQuery: "param=value",
			},
			Type: "url",
		}},
	})
	completion.Header.Set("foo", "bar")
	completion.Header.Set(HeaderOperationID, "test-operation-id")
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
	errorChecker func(error) error
}

func (h *failureExpectingCompletionHandler) CompleteOperation(ctx context.Context, completion *CompletionRequest) error {
	if completion.State != OperationStateCanceled {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "unexpected completion state: %q", completion.State)
	}
	if err := h.errorChecker(completion.Error); err != nil {
		return err
	}
	if completion.HTTPRequest.Header.Get("foo") != "bar" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid 'foo' header: %q", completion.HTTPRequest.Header.Get("foo"))
	}
	if completion.OperationID != "test-operation-id" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid %q header: %q", HeaderOperationID, completion.HTTPRequest.Header.Get(HeaderOperationID))
	}
	if len(completion.Links) == 0 {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "expected Links to be set on CompletionRequest")
	}

	return nil
}

func TestFailureCompletion(t *testing.T) {
	ctx, callbackURL, teardown := setupForCompletion(t, &failureExpectingCompletionHandler{
		errorChecker: func(err error) error {
			if err.Error() != "expected message" {
				return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid failure: %v", err)
			}
			return nil
		},
	}, nil, nil)
	defer teardown()

	completion, err := NewOperationCompletionUnsuccessful(NewCanceledOperationError(errors.New("expected message")), OperationCompletionUnsuccessfulOptions{
		OperationID: "test-operation-id",
		StartTime:   time.Now(),
		Links: []Link{{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/something",
				RawQuery: "param=value",
			},
			Type: "url",
		}},
	})
	require.NoError(t, err)
	completion.Header.Set("foo", "bar")
	request, err := NewCompletionHTTPRequest(ctx, callbackURL, completion)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
}

func TestFailureCompletion_CustomFailureConverter(t *testing.T) {
	fc := customFailureConverter{}
	ctx, callbackURL, teardown := setupForCompletion(t, &failureExpectingCompletionHandler{
		errorChecker: func(err error) error {
			if !errors.Is(err, errCustom) {
				return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid failure, expected a custom error: %v", err)
			}
			return nil
		},
	}, nil, fc)
	defer teardown()

	completion, err := NewOperationCompletionUnsuccessful(NewCanceledOperationError(errors.New("expected message")), OperationCompletionUnsuccessfulOptions{
		FailureConverter: fc,
		OperationID:      "test-operation-id",
		StartTime:        time.Now(),
		Links: []Link{{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/something",
				RawQuery: "param=value",
			},
			Type: "url",
		}},
	})
	require.NoError(t, err)
	completion.Header.Set("foo", "bar")
	request, err := NewCompletionHTTPRequest(ctx, callbackURL, completion)
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
	ctx, callbackURL, teardown := setupForCompletion(t, &failingCompletionHandler{}, nil, nil)
	defer teardown()

	completion, err := NewOperationCompletionSuccessful([]byte("success"), OperationCompletionSuccessfulOptions{})
	require.NoError(t, err)
	request, err := NewCompletionHTTPRequest(ctx, callbackURL, completion)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}
