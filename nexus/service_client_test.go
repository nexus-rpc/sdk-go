package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPClient(t *testing.T) {
	var err error

	transport, err := NewHTTPTransport(HTTPTransportOptions{BaseURL: "http://example.com"})
	require.NoError(t, err)

	_, err = NewServiceClient(ServiceClientOptions{Service: "", Transport: transport})
	require.ErrorContains(t, err, "empty Service")

	_, err = NewServiceClient(ServiceClientOptions{Service: "valid", Transport: nil})
	require.ErrorContains(t, err, "nil Transport")

	_, err = NewServiceClient(ServiceClientOptions{Service: "valid", Transport: transport})
	require.NoError(t, err)
}
