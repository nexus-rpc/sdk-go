package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexusclient"
	"github.com/nexus-rpc/sdk-go/nexusserver"
	"github.com/stretchr/testify/require"
)

const testTimeout = time.Second * 5

func setup(t *testing.T, handler nexusserver.Handler) (ctx context.Context, client *nexusclient.Client, teardown func()) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)

	httpHandler := nexusserver.NewHTTPHandler(nexusserver.Options{
		Handler: handler,
	})

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	client, err = nexusclient.NewClient(nexusclient.Options{
		GetResultMaxRequestTimeout: time.Minute,
		ServiceBaseURL:             fmt.Sprintf("http://%s/", listener.Addr().String()),
	})
	require.NoError(t, err)

	go func() {
		if err := http.Serve(listener, httpHandler); err != nil {
			log.Panicf("serve failed: %s", err)
		}
	}()

	return ctx, client, func() {
		cancel()
		listener.Close()
	}
}

func setupForCompletion(t *testing.T, handler nexusserver.CompletionHandler) (ctx context.Context, client *nexusclient.Client, callbackURL string, teardown func()) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)

	httpHandler := nexusserver.NewCompletionHTTPHandler(nexusserver.CompletionOptions{
		Handler: handler,
	})

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	callbackURL = fmt.Sprintf("http://%s/callback?a=b", listener.Addr().String())

	client, err = nexusclient.NewClient(nexusclient.Options{})
	require.NoError(t, err)

	go func() {
		if err := http.Serve(listener, httpHandler); err != nil {
			log.Panicf("serve failed: %s", err)
		}
	}()

	return ctx, client, callbackURL, func() {
		cancel()
		listener.Close()
	}
}
