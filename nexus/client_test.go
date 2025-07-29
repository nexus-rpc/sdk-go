package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPClient(t *testing.T) {
	var err error

	transport, err := NewHTTPTransport(HTTPTransportOptions{BaseURL: "http://example.com"})
	require.NoError(t, err)

	_, err = NewClient(ClientOptions{Service: "", Transport: transport})
	require.ErrorContains(t, err, "empty Service")

	_, err = NewClient(ClientOptions{Service: "valid", Transport: nil})
	require.ErrorContains(t, err, "nil Transport")

	_, err = NewClient(ClientOptions{Service: "valid", Transport: transport})
	require.NoError(t, err)
}
