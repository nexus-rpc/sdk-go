package nexus

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHandlerErrorfPreservesCauses(t *testing.T) {
	cause := fmt.Errorf("much goodness needed")
	err := NewHandlerErrorf(HandlerErrorTypeBadRequest, "insufficient goodness in request: %w", cause)
	require.Equal(t, "handler error (BAD_REQUEST): insufficient goodness in request: much goodness needed", err.Error())
	require.Equal(t, cause, err.Unwrap())
}
