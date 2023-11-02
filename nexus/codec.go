package nexus

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// A Stream is a container for a header and a reader.
// It is used to stream inputs and outputs in the various client and server APIs.
type Stream struct {
	// Header that should include information on how to decode this stream.
	// Headers constructed by the framework always have lower case keys.
	// User provided keys are considered case-insensitive by the framework.
	Header map[string]string
	// Reader contains request or response data. May be nil for empty data.
	Reader io.ReadCloser
}

// A LazyValue holds a value encoded in an underlying [Stream].
//
// ⚠️ When a LazyValue is returned from a client - if directly accessing the stream - it must be read it in its entirety
// and closed to free up the associated HTTP connection. Otherwise the [LazyValue.Consume] method must be called.
//
// ⚠️ When a LazyValue is passed to a server handler, it must not be used after the returning from the handler method.
type LazyValue struct {
	codec  Codec
	Stream *Stream
}

// Consume consumes the lazy value, decodes it from the underlying stream, and stores the result in the value pointed to by v.
//
//	var v int
//	err := s.Consume(&v)
func (s *LazyValue) Consume(v any) error {
	defer s.Stream.Reader.Close()
	return s.codec.Decode(s.Stream, v)
}

// Codec is used by the framework to serialize/deserialize input and output.
// To customize serialization logic, implement this interface and provide your implementation to framework methods such
// as [NewClient] and [NewHTTPHandler].
// By default, the SDK supports serialization of JSONables, byte slices, and nils.
type Codec interface {
	// Encode encodes a value into a [Stream].
	Encode(any) (*Stream, error)
	// Decode decodes a [Stream] into a given reference.
	Decode(*Stream, any) error
}

type codecChain []Codec

var errCodecIncompatible = errors.New("incompatible codec")

func (c codecChain) Encode(v any) (*Stream, error) {
	for _, l := range c {
		p, err := l.Encode(v)
		if err != nil {
			if errors.Is(err, errCodecIncompatible) {
				continue
			}
			return nil, err
		}
		return p, nil
	}
	return nil, errCodecIncompatible
}

func (c codecChain) Decode(s *Stream, v any) error {
	lenc := len(c)
	for i := range c {
		l := c[lenc-i-1]
		if err := l.Decode(s, v); err != nil {
			if errors.Is(err, errCodecIncompatible) {
				continue
			}
			return err
		}
		return nil
	}
	return errCodecIncompatible
}

var _ Codec = codecChain{}

type jsonCodec struct{}

func (jsonCodec) Decode(s *Stream, v any) error {
	if !isMediaTypeJSON(s.Header["content-type"]) {
		return errCodecIncompatible
	}
	body, err := io.ReadAll(s.Reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, &v)
}

func (jsonCodec) Encode(v any) (*Stream, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &Stream{
		Header: map[string]string{
			"content-type":   "application/json",
			"content-length": fmt.Sprintf("%d", len(body)),
		},
		Reader: io.NopCloser(bytes.NewReader(body)),
	}, nil
}

var _ Codec = jsonCodec{}

type nilCodec struct{}

func (nilCodec) Decode(s *Stream, v any) error {
	if s.Header["content-length"] != "0" {
		return errCodecIncompatible
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("could not decode non pointer value: %v", v)
	}
	re := rv.Elem()
	if !re.CanSet() {
		return fmt.Errorf("non settable type: %v", v)
	}
	// Set the zero value for the given type.
	re.Set(reflect.Zero(re.Type()))

	return nil
}

func (nilCodec) Encode(v any) (*Stream, error) {
	if v != nil {
		rv := reflect.ValueOf(v)
		if !(rv.Kind() == reflect.Ptr && rv.IsNil()) {
			return nil, errCodecIncompatible
		}
	}
	return &Stream{
		Header: map[string]string{"content-length": "0"},
		Reader: nil,
	}, nil
}

var _ Codec = nilCodec{}

type byteSliceCodec struct{}

func (byteSliceCodec) Decode(s *Stream, v any) error {
	if !isMediaTypeOctetStream(s.Header["content-type"]) {
		return errCodecIncompatible
	}
	if bPtr, ok := v.(*[]byte); ok {
		if bPtr == nil {
			return fmt.Errorf("cannot decode into nil pointer: %v", v)
		}
		b, err := io.ReadAll(s.Reader)
		if err != nil {
			return err
		}
		*bPtr = b
		return nil
	}
	// v is *any
	rv := reflect.ValueOf(v).Elem()
	if rv.Kind() == reflect.Interface {
		b, err := io.ReadAll(s.Reader)
		if err != nil {
			return err
		}
		rv.Set(reflect.ValueOf(b))
		return nil
	}
	return fmt.Errorf("unsupported value type for content: %v", v)
}

func (byteSliceCodec) Encode(v any) (*Stream, error) {
	if b, ok := v.([]byte); ok {
		return &Stream{
			Header: map[string]string{
				"content-type":   "application/octet-stream",
				"content-length": fmt.Sprintf("%d", len(b)),
			},
			Reader: io.NopCloser(bytes.NewReader(b)),
		}, nil
	}
	return nil, errCodecIncompatible
}

var _ Codec = byteSliceCodec{}

type compositeCodec struct {
	codecChain
}

var defaultCodec Codec = compositeCodec{
	codecChain([]Codec{nilCodec{}, byteSliceCodec{}, jsonCodec{}}),
}
