package nexus

import (
	"context"
	"errors"
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
	if completion.OperationID != "test-operation-token" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid operation ID: %q", completion.OperationID)
	}
	if completion.OperationToken != "test-operation-token" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid operation token: %q", completion.OperationToken)
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
	ctx, client, callbackURL, teardown := setupForCompletion(t, &successfulCompletionHandler{}, nil, nil)
	defer teardown()

	completeOpts := CompleteOperationOptions{
		Header:         Header{"foo": "bar"},
		Result:         666,
		OperationToken: "test-operation-token",
		StartTime:      time.Now(),
		Links: []Link{{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/something",
				RawQuery: "param=value",
			},
			Type: "url",
		}},
	}

	err := client.CompleteOperation(ctx, callbackURL, completeOpts)
	require.NoError(t, err)
}

func TestSuccessfulCompletion_CustomSerializer(t *testing.T) {
	serializer := &customSerializer{}
	ctx, client, callbackURL, teardown := setupForCompletion(t, &successfulCompletionHandler{}, serializer, nil)
	defer teardown()

	completeOpts := CompleteOperationOptions{
		Header: Header{"foo": "bar", HeaderOperationToken: "test-operation-token"},
		Result: 666,
		Links: []Link{{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/something",
				RawQuery: "param=value",
			},
			Type: "url",
		}},
	}

	err := client.CompleteOperation(ctx, callbackURL, completeOpts)
	require.NoError(t, err)

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
	if completion.OperationID != "test-operation-token" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid operation ID: %q", completion.OperationID)
	}
	if completion.OperationToken != "test-operation-token" {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid operation token: %q", completion.OperationToken)
	}
	if len(completion.Links) == 0 {
		return HandlerErrorf(HandlerErrorTypeBadRequest, "expected Links to be set on CompletionRequest")
	}

	return nil
}

func TestFailureCompletion(t *testing.T) {
	ctx, client, callbackURL, teardown := setupForCompletion(t, &failureExpectingCompletionHandler{
		errorChecker: func(err error) error {
			if err.Error() != "operation canceled: expected message" {
				return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid failure: %v", err)
			}
			return nil
		},
	}, nil, nil)
	defer teardown()

	completeOpts := CompleteOperationOptions{
		Header:         Header{"foo": "bar"},
		Error:          NewOperationCanceledError("expected message"),
		OperationToken: "test-operation-token",
		StartTime:      time.Now(),
		Links: []Link{{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/something",
				RawQuery: "param=value",
			},
			Type: "url",
		}},
	}

	err := client.CompleteOperation(ctx, callbackURL, completeOpts)
	require.NoError(t, err)
}

func TestFailureCompletion_CustomFailureConverter(t *testing.T) {
	fc := customFailureConverter{}
	ctx, client, callbackURL, teardown := setupForCompletion(t, &failureExpectingCompletionHandler{
		errorChecker: func(err error) error {
			if !errors.Is(err, errCustom) {
				return HandlerErrorf(HandlerErrorTypeBadRequest, "invalid failure, expected a custom error: %v", err)
			}
			return nil
		},
	}, nil, fc)
	defer teardown()

	completeOpts := CompleteOperationOptions{
		Header:         Header{"foo": "bar"},
		Error:          NewOperationCanceledError("expected message"),
		OperationToken: "test-operation-token",
		StartTime:      time.Now(),
		Links: []Link{{
			URL: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/something",
				RawQuery: "param=value",
			},
			Type: "url",
		}},
	}

	err := client.CompleteOperation(ctx, callbackURL, completeOpts)
	require.NoError(t, err)
}

type failingCompletionHandler struct {
}

func (h *failingCompletionHandler) CompleteOperation(ctx context.Context, completion *CompletionRequest) error {
	return HandlerErrorf(HandlerErrorTypeBadRequest, "I can't get no satisfaction")
}

func TestBadRequestCompletion(t *testing.T) {
	ctx, client, callbackURL, teardown := setupForCompletion(t, &failingCompletionHandler{}, nil, nil)
	defer teardown()

	completeOpts := CompleteOperationOptions{
		Result: []byte("success"),
	}

	err := client.CompleteOperation(ctx, callbackURL, completeOpts)
	var handlerErr *HandlerError
	require.ErrorAs(t, err, &handlerErr)
	require.Equal(t, HandlerErrorTypeBadRequest, handlerErr.Type)

	completeOpts.Error = NewOperationFailedError("some failure")
	err = client.CompleteOperation(ctx, callbackURL, completeOpts)
	require.ErrorIs(t, err, errResultAndErrorSet)
}
