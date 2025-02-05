package nexus_test

import (
	"context"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
)

func TestHandlerContext(t *testing.T) {
	ctx := nexus.WithHandlerContext(context.Background(), nexus.HandlerInfo{Operation: "test"})
	require.True(t, nexus.IsHandlerContext(ctx))
	initial := []nexus.Link{{Type: "foo"}, {Type: "bar"}}
	nexus.AddHandlerLinks(ctx, initial...)
	additional := nexus.Link{Type: "baz"}
	nexus.AddHandlerLinks(ctx, additional)
	require.Equal(t, append(initial, additional), nexus.HandlerLinks(ctx))
	nexus.SetHandlerLinks(ctx, initial...)
	require.Equal(t, initial, nexus.HandlerLinks(ctx))
	require.Equal(t, nexus.HandlerInfo{Operation: "test"}, nexus.ExtractHandlerInfo(ctx))
}
