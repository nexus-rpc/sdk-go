package nexus

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// A Content is a container for a header and a reader.
// It is used to stream inputs and outputs in the various client and server APIs.
type Content struct {
	// Header that should include information on how to deserialize this content.
	// Headers constructed by the framework always have lower case keys.
	// User provided keys are considered case-insensitive by the framework.
	Header Header
	// Reader contains request or response data. May be nil for empty data.
	Reader io.ReadCloser
}

// A LazyValue holds a value encoded in an underlying [Content].
//
// ⚠️ When a LazyValue is returned from a client - if directly accessing the content - it must be read it in its entirety
// and closed to free up the associated HTTP connection. Otherwise the [LazyValue.Consume] method must be called.
//
// ⚠️ When a LazyValue is passed to a server handler, it must not be used after the returning from the handler method.
type LazyValue struct {
	serializer Serializer
	Content    *Content
}

// Consume consumes the lazy value, decodes it from the underlying content, and stores the result in the value pointed to by v.
//
//	var v int
//	err := lazyValue.Consume(&v)
func (l *LazyValue) Consume(v any) error {
	defer l.Content.Reader.Close()
	return l.serializer.Deserialize(l.Content, v)
}

// Serializer is used by the framework to serialize/deserialize input and output.
// To customize serialization logic, implement this interface and provide your implementation to framework methods such
// as [NewClient] and [NewHTTPHandler].
// By default, the SDK supports serialization of JSONables, byte slices, and nils.
type Serializer interface {
	// Serialize encodes a value into a [Content].
	Serialize(any) (*Content, error)
	// Deserialize decodes a [Content] into a given reference.
	Deserialize(*Content, any) error
}

var anyType = reflect.TypeOf((*any)(nil)).Elem()

var errSerializerIncompatible = errors.New("incompatible serializer")

type serializerChain []Serializer

func (c serializerChain) Serialize(v any) (*Content, error) {
	for _, l := range c {
		p, err := l.Serialize(v)
		if err != nil {
			if errors.Is(err, errSerializerIncompatible) {
				continue
			}
			return nil, err
		}
		return p, nil
	}
	return nil, errSerializerIncompatible
}

func (c serializerChain) Deserialize(content *Content, v any) error {
	lenc := len(c)
	for i := range c {
		l := c[lenc-i-1]
		if err := l.Deserialize(content, v); err != nil {
			if errors.Is(err, errSerializerIncompatible) {
				continue
			}
			return err
		}
		return nil
	}
	return errSerializerIncompatible
}

var _ Serializer = serializerChain{}

type jsonSerializer struct{}

func (jsonSerializer) Deserialize(c *Content, v any) error {
	if !isMediaTypeJSON(c.Header["type"]) {
		return errSerializerIncompatible
	}
	body, err := io.ReadAll(c.Reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, &v)
}

func (jsonSerializer) Serialize(v any) (*Content, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &Content{
		Header: Header{
			"type":   "application/json",
			"length": fmt.Sprintf("%d", len(body)),
		},
		Reader: io.NopCloser(bytes.NewReader(body)),
	}, nil
}

var _ Serializer = jsonSerializer{}

type nilSerializer struct{}

func (nilSerializer) Deserialize(c *Content, v any) error {
	if c.Header["length"] != "0" {
		return errSerializerIncompatible
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("cannot deserialize into non pointer: %v", v)
	}
	if rv.IsNil() {
		return fmt.Errorf("cannot deserialize into nil pointer: %v", v)
	}
	re := rv.Elem()
	if !re.CanSet() {
		return fmt.Errorf("non settable type: %v", v)
	}
	// Set the zero value for the given type.
	re.Set(reflect.Zero(re.Type()))

	return nil
}

func (nilSerializer) Serialize(v any) (*Content, error) {
	if v != nil {
		rv := reflect.ValueOf(v)
		if !(rv.Kind() == reflect.Pointer && rv.IsNil()) {
			return nil, errSerializerIncompatible
		}
	}
	return &Content{
		Header: Header{"length": "0"},
		Reader: nil,
	}, nil
}

var _ Serializer = nilSerializer{}

type byteSliceSerializer struct{}

func (byteSliceSerializer) Deserialize(c *Content, v any) error {
	if !isMediaTypeOctetStream(c.Header["type"]) {
		return errSerializerIncompatible
	}
	if bPtr, ok := v.(*[]byte); ok {
		if bPtr == nil {
			return fmt.Errorf("cannot deserialize into nil pointer: %v", v)
		}
		b, err := io.ReadAll(c.Reader)
		if err != nil {
			return err
		}
		*bPtr = b
		return nil
	}
	// v is *any
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("cannot deserialize into non pointer: %v", v)
	}
	if rv.IsNil() {
		return fmt.Errorf("cannot deserialize into nil pointer: %v", v)
	}
	if rv.Elem().Type() != anyType {
		return fmt.Errorf("unsupported value type for content: %v", v)
	}
	b, err := io.ReadAll(c.Reader)
	if err != nil {
		return err
	}
	rv.Elem().Set(reflect.ValueOf(b))
	return nil
}

func (byteSliceSerializer) Serialize(v any) (*Content, error) {
	if b, ok := v.([]byte); ok {
		return &Content{
			Header: Header{
				"type":   "application/octet-stream",
				"length": fmt.Sprintf("%d", len(b)),
			},
			Reader: io.NopCloser(bytes.NewReader(b)),
		}, nil
	}
	return nil, errSerializerIncompatible
}

var _ Serializer = byteSliceSerializer{}

type compositeSerializer struct {
	serializerChain
}

var defaultSerializer Serializer = compositeSerializer{
	serializerChain([]Serializer{nilSerializer{}, byteSliceSerializer{}, jsonSerializer{}}),
}
