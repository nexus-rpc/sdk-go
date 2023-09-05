package nexus

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testTimeout = time.Second * 5
const getResultMaxTimeout = time.Millisecond * 300

func setup(t *testing.T, handler Handler) (ctx context.Context, client *Client, teardown func()) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)

	httpHandler := NewHTTPHandler(HandlerOptions{
		GetResultTimeout: getResultMaxTimeout,
		Handler:          handler,
	})

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	client, err = NewClient(ClientOptions{
		ServiceBaseURL: fmt.Sprintf("http://%s/", listener.Addr().String()),
	})
	require.NoError(t, err)

	go func() {
		// Ignore for test purposes
		_ = http.Serve(listener, httpHandler)
	}()

	return ctx, client, func() {
		cancel()
		listener.Close()
	}
}

func setupForCompletion(t *testing.T, handler CompletionHandler) (ctx context.Context, callbackURL string, teardown func()) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)

	httpHandler := NewCompletionHTTPHandler(CompletionHandlerOptions{
		Handler: handler,
	})

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	callbackURL = fmt.Sprintf("http://%s/callback?a=b", listener.Addr().String())

	go func() {
		// Ignore for test purposes
		_ = http.Serve(listener, httpHandler)
	}()

	return ctx, callbackURL, func() {
		cancel()
		listener.Close()
	}
}
