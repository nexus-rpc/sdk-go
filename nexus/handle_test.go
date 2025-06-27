package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHandleFailureConditions(t *testing.T) {
	client, err := NewHTTPClient(HTTPClientOptions{ClientOptions: ClientOptions{BaseURL: "http://foo.com", Service: "test"}})
	require.NoError(t, err)
	_, err = client.NewHandle("", "token")
	require.ErrorIs(t, err, errEmptyOperationName)
	_, err = client.NewHandle("name", "")
	require.ErrorIs(t, err, errEmptyOperationToken)
	_, err = client.NewHandle("", "")
	require.ErrorIs(t, err, errEmptyOperationName)
	require.ErrorIs(t, err, errEmptyOperationToken)
}
