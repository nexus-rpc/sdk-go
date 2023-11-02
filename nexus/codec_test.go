package nexus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONCodec(t *testing.T) {
	var err error
	var stream *Stream
	c := jsonCodec{}
	stream, err = c.Encode(1)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"content-type": "application/json", "content-length": "1"}, stream.Header)
	var i int
	err = c.Decode(stream, &i)
	require.NoError(t, err)
	require.Equal(t, 1, i)
}

func TestNilCodec(t *testing.T) {
	var err error
	var stream *Stream
	c := nilCodec{}
	_, err = c.Encode(1)
	require.ErrorIs(t, err, errCodecIncompatible)

	stream, err = c.Encode(nil)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"content-length": "0"}, stream.Header)
	var out any
	require.NoError(t, c.Decode(stream, &out))
	require.Equal(t, nil, out)

	type s struct{ Member string }
	// struct is set to zero value
	structIn := s{Member: "gold"}
	require.NoError(t, c.Decode(stream, &structIn))
	require.Equal(t, s{}, structIn)

	// nil pointer
	type NoValue *struct{}
	var nv NoValue

	stream, err = c.Encode(nv)
	require.NoError(t, err)
	require.NoError(t, c.Decode(stream, &nv))
	require.Equal(t, NoValue(nil), nv)
}

func TestByteSliceCodec(t *testing.T) {
	var err error
	var stream *Stream
	c := byteSliceCodec{}
	_, err = c.Encode(1)
	require.ErrorIs(t, err, errCodecIncompatible)

	// decode into byte slice
	stream, err = c.Encode([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"content-type": "application/octet-stream", "content-length": "3"}, stream.Header)
	var out []byte
	require.NoError(t, c.Decode(stream, &out))
	require.Equal(t, []byte("abc"), out)

	stream, err = c.Encode([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"content-type": "application/octet-stream", "content-length": "3"}, stream.Header)
	// decode into nil pointer fails
	var pout *[]byte
	require.ErrorContains(t, c.Decode(stream, pout), "cannot decode into nil pointer")
	// decode into any
	var aout any
	require.NoError(t, c.Decode(stream, &aout))
	require.Equal(t, []byte("abc"), aout)
}

func TestDefaultCodec(t *testing.T) {
	var err error
	var stream *Stream
	c := defaultCodec

	// JSON
	var i int
	stream, err = c.Encode(1)
	require.NoError(t, err)
	require.NoError(t, c.Decode(stream, &i))
	require.Equal(t, 1, i)

	// byte slice
	var b []byte
	stream, err = c.Encode([]byte("abc"))
	require.NoError(t, err)
	require.NoError(t, c.Decode(stream, &b))
	require.Equal(t, []byte("abc"), b)

	// nil
	var a any
	stream, err = c.Encode(nil)
	require.NoError(t, err)
	require.NoError(t, c.Decode(stream, &a))
	require.Equal(t, nil, a)
}
