package nexus

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceBaseURL(t *testing.T) {
	var err error

	_, err = New(Options{ServiceBaseURL: "-http://invalid"})
	var urlError *url.Error
	require.ErrorAs(t, err, &urlError)

	_, err = New(Options{ServiceBaseURL: "smtp://example.com"})
	require.ErrorIs(t, err, errInvalidURLScheme)

	_, err = New(Options{ServiceBaseURL: "http://example.com"})
	require.NoError(t, err)

	_, err = New(Options{ServiceBaseURL: "https://example.com"})
	require.NoError(t, err)
}

func TestGetResultMaxRequestTimeout(t *testing.T) {
	var err error
	var client *Client

	client, err = New(Options{
		ServiceBaseURL:      "http://unit.test",
		GetResultMaxTimeout: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, time.Second, client.Options.GetResultMaxTimeout)

	// Default is set
	client, err = New(Options{
		ServiceBaseURL: "http://unit.test",
	})
	require.NoError(t, err)
	require.Equal(t, time.Minute, client.Options.GetResultMaxTimeout)
}
