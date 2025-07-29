package nexus

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPTransport(t *testing.T) {
	var err error

	_, err = NewHTTPTransport(HTTPTransportOptions{BaseURL: ""})
	require.ErrorContains(t, err, "empty BaseURL")

	_, err = NewHTTPTransport(HTTPTransportOptions{BaseURL: "-http://invalid"})
	var urlError *url.Error
	require.ErrorAs(t, err, &urlError)

	_, err = NewHTTPTransport(HTTPTransportOptions{BaseURL: "smtp://example.com"})
	require.ErrorContains(t, err, "invalid URL scheme: smtp")

	_, err = NewHTTPTransport(HTTPTransportOptions{BaseURL: "http://example.com"})
	require.NoError(t, err)
}
