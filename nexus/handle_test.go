package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHandleFailureConditions(t *testing.T) {
	tr, err := NewHTTPTransport(HTTPTransportOptions{BaseURL: "http://foo.com"})
	require.NoError(t, err)

	client, err := NewServiceClient(ServiceClientOptions{Service: "test", Transport: tr})
	require.NoError(t, err)
	_, err = client.NewOperationHandle("", "token")
	require.ErrorIs(t, err, errEmptyOperationName)
	_, err = client.NewOperationHandle("name", "")
	require.ErrorIs(t, err, errEmptyOperationToken)
	_, err = client.NewOperationHandle("", "")
	require.ErrorIs(t, err, errEmptyOperationName)
	require.ErrorIs(t, err, errEmptyOperationToken)
}
