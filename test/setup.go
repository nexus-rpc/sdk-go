package test

import (
	"context"
	"fmt"
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

	go http.Serve(listener, httpHandler)

	return ctx, client, func() {
		cancel()
		listener.Close()
	}
}
