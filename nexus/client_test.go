package nexus

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	var err error

	_, err = NewHTTPClient(HTTPClientOptions{ClientOptions: ClientOptions{BaseURL: "", Service: "ignored"}})
	require.ErrorContains(t, err, "empty BaseURL")

	_, err = NewHTTPClient(HTTPClientOptions{ClientOptions: ClientOptions{BaseURL: "-http://invalid", Service: "ignored"}})
	var urlError *url.Error
	require.ErrorAs(t, err, &urlError)

	_, err = NewHTTPClient(HTTPClientOptions{ClientOptions: ClientOptions{BaseURL: "smtp://example.com", Service: "ignored"}})
	require.ErrorContains(t, err, "invalid URL scheme: smtp")

	_, err = NewHTTPClient(HTTPClientOptions{ClientOptions: ClientOptions{BaseURL: "http://example.com", Service: "ignored"}})
	require.NoError(t, err)

	_, err = NewHTTPClient(HTTPClientOptions{ClientOptions: ClientOptions{BaseURL: "https://example.com", Service: ""}})
	require.ErrorContains(t, err, "empty Service")

	_, err = NewHTTPClient(HTTPClientOptions{ClientOptions: ClientOptions{BaseURL: "https://example.com", Service: "valid"}})
	require.NoError(t, err)
}
