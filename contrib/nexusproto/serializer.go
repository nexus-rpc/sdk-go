package nexusproto

import (
	"fmt"
	"mime"

	"github.com/nexus-rpc/sdk-go/nexus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func isProtoJSON(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json" && len(params) == 2 && params["format"] == "protobuf" && params["message-type"] != ""
}

func isProtoBinary(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/x-protobuf" && len(params) == 1 && params["message-type"] != ""
}

type protoJsonSerializer struct{}

func (protoJsonSerializer) Deserialize(c *nexus.Content, v any) error {
	if !isProtoJSON(c.Header["type"]) {
		return nexus.ErrSerializerIncompatible
	}
	msg, ok := v.(proto.Message)
	if !ok {
		return nexus.ErrSerializerIncompatible
	}
	return protojson.Unmarshal(c.Data, msg)
}

func (protoJsonSerializer) Serialize(v any) (*nexus.Content, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, nexus.ErrSerializerIncompatible
	}
	data, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err
	}
	messageType := string(msg.ProtoReflect().Descriptor().FullName())
	return &nexus.Content{
		Header: nexus.Header{
			"type": fmt.Sprintf("application/json; format=protobuf; message-type=%q", messageType),
		},
		Data: data,
	}, nil
}

type protoBinarySerializer struct{}

func (protoBinarySerializer) Deserialize(c *nexus.Content, v any) error {
	if !isProtoBinary(c.Header["type"]) {
		return nexus.ErrSerializerIncompatible
	}
	msg, ok := v.(proto.Message)
	if !ok {
		return nexus.ErrSerializerIncompatible
	}
	return proto.Unmarshal(c.Data, msg)
}

func (protoBinarySerializer) Serialize(v any) (*nexus.Content, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, nexus.ErrSerializerIncompatible
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	messageType := string(msg.ProtoReflect().Descriptor().FullName())
	return &nexus.Content{
		Header: nexus.Header{
			"type": fmt.Sprintf("application/x-protobuf; message-type=%q", messageType),
		},
		Data: data,
	}, nil
}

type SerializerMode int

const (
	SerializerModePreferJSON = SerializerMode(iota)
	SerializerModePreferBinary
)

func Serializer(mode SerializerMode) nexus.Serializer {
	serializers := []nexus.Serializer{
		nexus.NilSerializer{},
	}
	if mode == SerializerModePreferJSON {
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
