package nexus

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONSerializer(t *testing.T) {
	var err error
	var c *Content
	s := jsonSerializer{}
	c, err = s.Serialize(1)
	require.NoError(t, err)
	require.Equal(t, Header{"type": "application/json"}, c.Header)
	var i int
	err = s.Deserialize(c, &i)
	require.NoError(t, err)
	require.Equal(t, 1, i)
}

func TestNilSerializer(t *testing.T) {
	var err error
	var c *Content
	s := NilSerializer{}
	_, err = s.Serialize(1)
	require.ErrorIs(t, err, ErrSerializerIncompatible)

	c, err = s.Serialize(nil)
	require.NoError(t, err)
	require.Equal(t, Header{}, c.Header)
	var out any
	require.NoError(t, s.Deserialize(c, &out))
	require.Equal(t, nil, out)

	type st struct{ Member string }
	// struct is set to zero value
	structIn := st{Member: "gold"}
	require.NoError(t, s.Deserialize(c, &structIn))
	require.Equal(t, st{}, structIn)

	// nil pointer
	type NoValue *struct{}
	var nv NoValue

	c, err = s.Serialize(nv)
	require.NoError(t, err)
	require.NoError(t, s.Deserialize(c, &nv))
	require.Equal(t, NoValue(nil), nv)
}

func TestByteSliceSerializer(t *testing.T) {
	var err error
	var c *Content
	s := byteSliceSerializer{}
	_, err = s.Serialize(1)
	require.ErrorIs(t, err, ErrSerializerIncompatible)

	// decode into byte slice
	c, err = s.Serialize([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, Header{"type": "application/octet-stream"}, c.Header)
	var out []byte
	require.NoError(t, s.Deserialize(c, &out))
	require.Equal(t, []byte("abc"), out)

	c, err = s.Serialize([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, Header{"type": "application/octet-stream"}, c.Header)
	// decode into nil pointer fails
	var pout *[]byte
	require.ErrorContains(t, s.Deserialize(c, pout), "cannot deserialize into nil pointer")
	// decode into any
	var aout any
	require.NoError(t, s.Deserialize(c, &aout))
	require.Equal(t, []byte("abc"), aout)
}

func TestDefaultSerializer(t *testing.T) {
	var err error
	var c *Content
	s := defaultSerializer

	// JSON
	var i int
	c, err = s.Serialize(1)
	require.NoError(t, err)
	require.NoError(t, s.Deserialize(c, &i))
	require.Equal(t, 1, i)

	// byte slice
	var b []byte
	c, err = s.Serialize([]byte("abc"))
	require.NoError(t, err)
	require.NoError(t, s.Deserialize(c, &b))
	require.Equal(t, []byte("abc"), b)

	// nil
	var a any
	c, err = s.Serialize(nil)
	require.NoError(t, err)
	require.NoError(t, s.Deserialize(c, &a))
	require.Equal(t, nil, a)
}

// There's zero chance of concurrent updates in the test where this is used. Don't bother locking.
type customSerializer struct {
	encoded int
	decoded int
}

func (c *customSerializer) Serialize(v any) (*Content, error) {
	vint := v.(int)
	c.encoded++
	return &Content{
		Header: map[string]string{
			"custom": strconv.Itoa(vint),
		},
	}, nil
}

func (c *customSerializer) Deserialize(s *Content, v any) error {
	vintPtr := v.(*int)
	decoded, err := strconv.Atoi(s.Header["custom"])
	if err != nil {
		return err
	}
	*vintPtr = decoded
	c.decoded++
	return nil
}

func TestCustomSerializer(t *testing.T) {
	svc := NewService(testService)
	registry := NewServiceRegistry()
	require.NoError(t, svc.Register(
		numberValidatorOperation,
		asyncNumberValidatorOperationInstance,
	))
	require.NoError(t, registry.Register(svc))
	handler, err := registry.NewHandler()
	require.NoError(t, err)

	c := &customSerializer{}
	ctx, client, teardown := setupCustom(t, handler, c, nil)
	defer teardown()

	result, err := ExecuteOperation(ctx, client, numberValidatorOperation, 3, ExecuteOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, result)

	// Async triggers GetResult, test this too.
	result, err = ExecuteOperation(ctx, client, asyncNumberValidatorOperationInstance, 3, ExecuteOperationOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, result)

	require.Equal(t, 4, c.decoded)
	require.Equal(t, 4, c.encoded)
}

func TestDefaultFailureConverterArbitraryError(t *testing.T) {
	sourceErr := errors.New("test")
	var f Failure
	conv := defaultFailureConverter

	f = conv.ErrorToFailure(sourceErr)
	convErr := conv.FailureToError(f)
	require.Equal(t, sourceErr.Error(), convErr.Error())
}

func TestDefaultFailureConverterFailureError(t *testing.T) {
	sourceErr := &FailureError{
		Failure: Failure{
			Message:  "test",
			Metadata: map[string]string{"key": "value"},
			Details:  []byte(`"details"`),
		},
	}
	var f Failure
	conv := defaultFailureConverter

	f = conv.ErrorToFailure(sourceErr)
	convErr := conv.FailureToError(f)
	require.Equal(t, sourceErr, convErr)
}

type customFailureConverter struct{}

var errCustom = errors.New("custom")

// ErrorToFailure implements FailureConverter.
func (c customFailureConverter) ErrorToFailure(err error) Failure {
	return Failure{
		Message: err.Error(),
		Metadata: map[string]string{
			"type": "custom",
		},
	}
}

// FailureToError implements FailureConverter.
func (c customFailureConverter) FailureToError(f Failure) error {
	if f.Metadata["type"] != "custom" {
		return errors.New(f.Message)
	}
	return fmt.Errorf("%w: %s", errCustom, f.Message)
}

func TestCustomFailureConverter(t *testing.T) {
	svc := NewService(testService)
	registry := NewServiceRegistry()
	require.NoError(t, svc.Register(
		numberValidatorOperation,
		asyncNumberValidatorOperationInstance,
	))
	require.NoError(t, registry.Register(svc))
	handler, err := registry.NewHandler()
	require.NoError(t, err)

	c := customFailureConverter{}
	ctx, client, teardown := setupCustom(t, handler, nil, c)
	defer teardown()

	_, err = ExecuteOperation(ctx, client, numberValidatorOperation, 0, ExecuteOperationOptions{})
	require.ErrorIs(t, err, errCustom)

	// Async triggers GetResult, test this too.
	_, err = ExecuteOperation(ctx, client, asyncNumberValidatorOperationInstance, 0, ExecuteOperationOptions{})
	require.ErrorIs(t, err, errCustom)
}
