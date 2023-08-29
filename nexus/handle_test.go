package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHandleFailureConditions(t *testing.T) {
	client, err := NewClient(ClientOptions{})
	require.NoError(t, err)
	_, err = client.NewHandle("name", "id")
	require.ErrorIs(t, err, errEmptyServiceBaseURL)

	client, err = NewClient(ClientOptions{ServiceBaseURL: "http://foo.com"})
	require.NoError(t, err)
	_, err = client.NewHandle("", "id")
	require.ErrorIs(t, err, errInvalidOperationName)
	_, err = client.NewHandle("name", "")
	require.ErrorIs(t, err, errInvalidOperationID)
	_, err = client.NewHandle("", "")
	require.ErrorIs(t, err, errInvalidOperationName)
	require.ErrorIs(t, err, errInvalidOperationID)
}
