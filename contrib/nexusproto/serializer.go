package nexusproto

import (
	"fmt"
	"mime"
	"reflect"

	"github.com/nexus-rpc/sdk-go/nexus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func messageFromAny(v any) (proto.Message, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("invalid protobuf object: type: %T", v)
	}
	elem := rv.Type().Elem()

	// v is a proto.Message or *proto.Message, the serializer supports either:
	// 1. var d durationpb.Duration; s.Deserialize(&d)
	// 2. var d *durationpb.Duration; s.Deserialize(&d)
	// In the second case, we need to instantiate an empty proto struct:
	if elem.Kind() == reflect.Ptr {
		empty := reflect.New(elem.Elem())
		rv.Elem().Set(empty)
		rv = empty
	}

	msg, ok := rv.Interface().(proto.Message)
	if !ok {
		return nil, fmt.Errorf("%w: value is not a proto.Message or a pointer to one", nexus.ErrSerializerIncompatible)
	}

	return msg, nil
}

type protoJsonSerializer struct{}

func (protoJsonSerializer) extractMessageType(contentType string) string {
	if contentType == "" {
		return ""
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" || len(params) != 2 || params["format"] != "protobuf" {
		return ""
	}
	return params["message-type"]
}

func (s protoJsonSerializer) Deserialize(c *nexus.Content, v any) error {
	messageTypeFromHeader := s.extractMessageType(c.Header["type"])
	if messageTypeFromHeader == "" {
		return fmt.Errorf("%w: incompatible content type header", nexus.ErrSerializerIncompatible)
	}
	msg, err := messageFromAny(v)
	if err != nil {
		return err
	}
	messageTypeFromValue := string(msg.ProtoReflect().Type().Descriptor().FullName())
	if messageTypeFromHeader != messageTypeFromValue {
		return fmt.Errorf("serialized message type: %q is different from the provided value: %q", messageTypeFromHeader, messageTypeFromValue)

	}
	return protojson.Unmarshal(c.Data, msg)
}

func (protoJsonSerializer) Serialize(v any) (*nexus.Content, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("%w: value is not a proto.Message", nexus.ErrSerializerIncompatible)
	}
	data, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err
	}
	messageType := string(msg.ProtoReflect().Descriptor().FullName())
	return &nexus.Content{
		Header: nexus.Header{
			"type": mime.FormatMediaType("application/json", map[string]string{
				"format":       "protobuf",
				"message-type": messageType,
			}),
		},
		Data: data,
	}, nil
}

type protoBinarySerializer struct{}

func (protoBinarySerializer) extractMessageType(contentType string) string {
	if contentType == "" {
		return ""
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/x-protobuf" || len(params) != 1 {
		return ""
	}
	return params["message-type"]
}

func (s protoBinarySerializer) Deserialize(c *nexus.Content, v any) error {
	messageTypeFromHeader := s.extractMessageType(c.Header["type"])
	if messageTypeFromHeader == "" {
		return fmt.Errorf("%w: incompatible content type header", nexus.ErrSerializerIncompatible)
	}
	msg, err := messageFromAny(v)
	if err != nil {
		return err
	}
	messageTypeFromValue := string(msg.ProtoReflect().Type().Descriptor().FullName())
	if messageTypeFromHeader != messageTypeFromValue {
		return fmt.Errorf("serialized message type: %q is different from the provided value: %q", messageTypeFromHeader, messageTypeFromValue)

	}
	return proto.Unmarshal(c.Data, msg)
}

func (protoBinarySerializer) Serialize(v any) (*nexus.Content, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("%w: value is not a proto.Message", nexus.ErrSerializerIncompatible)
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	messageType := string(msg.ProtoReflect().Descriptor().FullName())
	return &nexus.Content{
		Header: nexus.Header{
			"type": mime.FormatMediaType("application/x-protobuf", map[string]string{
				"message-type": messageType,
			}),
		},
		Data: data,
	}, nil
}

// SerializerMode controls the preferred serialization format.
type SerializerMode int

const (
	// SerializerModePreferJSON instructs the serializer to prefer to serialize in proto JSON format.
	SerializerModePreferJSON = SerializerMode(iota)
	// SerializerModePreferBinary instructs the serializer to prefer to serialize in proto binary format.
	SerializerModePreferBinary
)

type SerializerOptions struct {
	// Mode is the preferred mode for the serializer.
	// The serializer supports deserializing both JSON and binary formats, but will prefer to serialize in the given
	// format.
	Mode SerializerMode
}

// NewSerializer constructs a Protobuf [nexus.Serializer] with the given options.
// The returned serializer supports serializing nil and proto messages.
func NewSerializer(options SerializerOptions) nexus.Serializer {
	serializers := []nexus.Serializer{
		nexus.NilSerializer{},
	}
	if options.Mode == SerializerModePreferJSON {
		serializers = append(serializers,
			protoJsonSerializer{},
			protoBinarySerializer{},
		)
	} else {
		serializers = append(serializers,
			protoBinarySerializer{},
			protoJsonSerializer{},
		)
	}
	return nexus.CompositeSerializer(serializers)
}
