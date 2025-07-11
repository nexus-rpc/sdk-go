package nexus

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPClient(t *testing.T) {
	var err error

	_, err = NewHTTPClient(ClientOptions{Service: "ignored"}, HTTPTransportOptions{BaseURL: ""})
	require.ErrorContains(t, err, "empty BaseURL")

	_, err = NewHTTPClient(ClientOptions{Service: "ignored"}, HTTPTransportOptions{BaseURL: "-http://invalid"})
	var urlError *url.Error
	require.ErrorAs(t, err, &urlError)

	_, err = NewHTTPClient(ClientOptions{Service: "ignored"}, HTTPTransportOptions{BaseURL: "smtp://example.com"})
	require.ErrorContains(t, err, "invalid URL scheme: smtp")

	_, err = NewHTTPClient(ClientOptions{Service: "ignored"}, HTTPTransportOptions{BaseURL: "http://example.com"})
	require.NoError(t, err)

	_, err = NewHTTPClient(ClientOptions{Service: ""}, HTTPTransportOptions{BaseURL: "https://example.com"})
	require.ErrorContains(t, err, "empty Service")

	_, err = NewHTTPClient(ClientOptions{Service: "valid"}, HTTPTransportOptions{BaseURL: "https://example.com"})
	require.NoError(t, err)
}
