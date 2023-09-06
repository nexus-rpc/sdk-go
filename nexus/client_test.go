package nexus

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceBaseURL(t *testing.T) {
	var err error

	_, err = NewClient(ClientOptions{ServiceBaseURL: ""})
	require.ErrorIs(t, err, errEmptyServiceBaseURL)

	_, err = NewClient(ClientOptions{ServiceBaseURL: "-http://invalid"})
	var urlError *url.Error
	require.ErrorAs(t, err, &urlError)

	_, err = NewClient(ClientOptions{ServiceBaseURL: "smtp://example.com"})
	require.ErrorIs(t, err, errInvalidURLScheme)

	_, err = NewClient(ClientOptions{ServiceBaseURL: "http://example.com"})
	require.NoError(t, err)

	_, err = NewClient(ClientOptions{ServiceBaseURL: "https://example.com"})
	require.NoError(t, err)
}
