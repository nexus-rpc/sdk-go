package nexusclient

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceBaseURL(t *testing.T) {
	var err error

	_, err = NewClient(Options{})
	require.ErrorIs(t, err, ErrEmptyServiceBaseURL)

	_, err = NewClient(Options{ServiceBaseURL: "-http://invalid"})
	var urlError *url.Error
	require.ErrorAs(t, err, &urlError)

	_, err = NewClient(Options{ServiceBaseURL: "smtp://example.com"})
	require.ErrorIs(t, err, ErrInvalidURLScheme)

	_, err = NewClient(Options{ServiceBaseURL: "http://example.com"})
	require.NoError(t, err)

	_, err = NewClient(Options{ServiceBaseURL: "https://example.com"})
	require.NoError(t, err)
}

func TestGetResultMaxRequestTimeout(t *testing.T) {
	var err error
	var client *Client

	client, err = NewClient(Options{
		ServiceBaseURL:             "http://unit.test",
		GetResultMaxRequestTimeout: time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, time.Second, client.Options.GetResultMaxRequestTimeout)

	// Default is set
	client, err = NewClient(Options{
		ServiceBaseURL: "http://unit.test",
	})
	require.NoError(t, err)
	require.Equal(t, time.Minute, client.Options.GetResultMaxRequestTimeout)
}
