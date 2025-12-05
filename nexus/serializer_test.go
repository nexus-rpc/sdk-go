package nexus

import (
	"errors"
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

func TestDefaultFailureConverterArbitraryError(t *testing.T) {
	sourceErr := errors.New("test")
	conv := defaultFailureConverter

	f, err := conv.ErrorToFailure(sourceErr)
	require.NoError(t, err)
	convErr, err := conv.FailureToError(f)
	require.NoError(t, err)
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
	conv := defaultFailureConverter

	f, err := conv.ErrorToFailure(sourceErr)
	require.NoError(t, err)
	convErr, err := conv.FailureToError(f)
	require.NoError(t, err)
	require.Equal(t, sourceErr, convErr)
}
