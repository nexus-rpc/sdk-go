// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.12
// source: nexus/rpc/api/result/v1/result.proto

package result

import (
	v1 "github.com/nexus-rpc/sdk-go/nexusapi/common/v1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Succeeded struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Headers map[string]*v1.HeaderValues `protobuf:"bytes,1,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Body    []byte                      `protobuf:"bytes,2,opt,name=body,proto3" json:"body,omitempty"`
}

func (x *Succeeded) Reset() {
	*x = Succeeded{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nexus_rpc_api_result_v1_result_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Succeeded) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Succeeded) ProtoMessage() {}

func (x *Succeeded) ProtoReflect() protoreflect.Message {
	mi := &file_nexus_rpc_api_result_v1_result_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Succeeded.ProtoReflect.Descriptor instead.
func (*Succeeded) Descriptor() ([]byte, []int) {
	return file_nexus_rpc_api_result_v1_result_proto_rawDescGZIP(), []int{0}
}

func (x *Succeeded) GetHeaders() map[string]*v1.HeaderValues {
	if x != nil {
		return x.Headers
	}
	return nil
}

func (x *Succeeded) GetBody() []byte {
	if x != nil {
		return x.Body
	}
	return nil
}

type Failed struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Headers map[string]*v1.HeaderValues `protobuf:"bytes,1,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Body    []byte                      `protobuf:"bytes,2,opt,name=body,proto3" json:"body,omitempty"`
	// TODO: should we have built-in codes here? It could be useful when we translate to HTTP.
	Retryable bool `protobuf:"varint,3,opt,name=retryable,proto3" json:"retryable,omitempty"`
}

func (x *Failed) Reset() {
	*x = Failed{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nexus_rpc_api_result_v1_result_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Failed) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Failed) ProtoMessage() {}

func (x *Failed) ProtoReflect() protoreflect.Message {
	mi := &file_nexus_rpc_api_result_v1_result_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Failed.ProtoReflect.Descriptor instead.
func (*Failed) Descriptor() ([]byte, []int) {
	return file_nexus_rpc_api_result_v1_result_proto_rawDescGZIP(), []int{1}
}

func (x *Failed) GetHeaders() map[string]*v1.HeaderValues {
	if x != nil {
		return x.Headers
	}
	return nil
}

func (x *Failed) GetBody() []byte {
	if x != nil {
		return x.Body
	}
	return nil
}

func (x *Failed) GetRetryable() bool {
	if x != nil {
		return x.Retryable
	}
	return false
}

type Canceled struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Headers map[string]*v1.HeaderValues `protobuf:"bytes,1,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Details of the cancelation.
	// TODO: do we want a message as string here?
	// Do we need details at all?
	Body []byte `protobuf:"bytes,2,opt,name=body,proto3" json:"body,omitempty"`
}

func (x *Canceled) Reset() {
	*x = Canceled{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nexus_rpc_api_result_v1_result_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Canceled) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Canceled) ProtoMessage() {}

func (x *Canceled) ProtoReflect() protoreflect.Message {
	mi := &file_nexus_rpc_api_result_v1_result_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Canceled.ProtoReflect.Descriptor instead.
func (*Canceled) Descriptor() ([]byte, []int) {
	return file_nexus_rpc_api_result_v1_result_proto_rawDescGZIP(), []int{2}
}

func (x *Canceled) GetHeaders() map[string]*v1.HeaderValues {
	if x != nil {
		return x.Headers
	}
	return nil
}

func (x *Canceled) GetBody() []byte {
	if x != nil {
		return x.Body
	}
	return nil
}

type Result struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Variant:
	//
	//	*Result_Succeeded
	//	*Result_Failed
	//	*Result_Canceled
	Variant isResult_Variant `protobuf_oneof:"variant"`
}

func (x *Result) Reset() {
	*x = Result{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nexus_rpc_api_result_v1_result_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Result) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Result) ProtoMessage() {}

func (x *Result) ProtoReflect() protoreflect.Message {
	mi := &file_nexus_rpc_api_result_v1_result_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Result.ProtoReflect.Descriptor instead.
func (*Result) Descriptor() ([]byte, []int) {
	return file_nexus_rpc_api_result_v1_result_proto_rawDescGZIP(), []int{3}
}

func (m *Result) GetVariant() isResult_Variant {
	if m != nil {
		return m.Variant
	}
	return nil
}

func (x *Result) GetSucceeded() *Succeeded {
	if x, ok := x.GetVariant().(*Result_Succeeded); ok {
		return x.Succeeded
	}
	return nil
}

func (x *Result) GetFailed() *Failed {
	if x, ok := x.GetVariant().(*Result_Failed); ok {
		return x.Failed
	}
	return nil
}

func (x *Result) GetCanceled() *Canceled {
	if x, ok := x.GetVariant().(*Result_Canceled); ok {
		return x.Canceled
	}
	return nil
}

type isResult_Variant interface {
	isResult_Variant()
}

type Result_Succeeded struct {
	Succeeded *Succeeded `protobuf:"bytes,2,opt,name=succeeded,proto3,oneof"`
}

type Result_Failed struct {
	Failed *Failed `protobuf:"bytes,3,opt,name=failed,proto3,oneof"`
}

type Result_Canceled struct {
	Canceled *Canceled `protobuf:"bytes,4,opt,name=canceled,proto3,oneof"`
}

func (*Result_Succeeded) isResult_Variant() {}

func (*Result_Failed) isResult_Variant() {}

func (*Result_Canceled) isResult_Variant() {}

var File_nexus_rpc_api_result_v1_result_proto protoreflect.FileDescriptor

var file_nexus_rpc_api_result_v1_result_proto_rawDesc = []byte{
	0x0a, 0x24, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x17, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70,
	0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x76, 0x31, 0x1a,
	0x24, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x63,
	0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xcd, 0x01, 0x0a, 0x09, 0x53, 0x75, 0x63, 0x63, 0x65, 0x65,
	0x64, 0x65, 0x64, 0x12, 0x49, 0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x2f, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x53,
	0x75, 0x63, 0x63, 0x65, 0x65, 0x64, 0x65, 0x64, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x12, 0x12,
	0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x62, 0x6f,
	0x64, 0x79, 0x1a, 0x61, 0x0a, 0x0c, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x6b, 0x65, 0x79, 0x12, 0x3b, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xe5, 0x01, 0x0a, 0x06, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64,
	0x12, 0x46, 0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x2c, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x61, 0x69, 0x6c,
	0x65, 0x64, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52,
	0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x12, 0x1c, 0x0a, 0x09,
	0x72, 0x65, 0x74, 0x72, 0x79, 0x61, 0x62, 0x6c, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x09, 0x72, 0x65, 0x74, 0x72, 0x79, 0x61, 0x62, 0x6c, 0x65, 0x1a, 0x61, 0x0a, 0x0c, 0x48, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65,
	0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x3b, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x6e, 0x65,
	0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6d, 0x6d,
	0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x73, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xcb, 0x01,
	0x0a, 0x08, 0x43, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x65, 0x64, 0x12, 0x48, 0x0a, 0x07, 0x68, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x6e, 0x65,
	0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75,
	0x6c, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x65, 0x64, 0x2e, 0x48,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x68, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x1a, 0x61, 0x0a, 0x0c, 0x48, 0x65, 0x61, 0x64,
	0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x3b, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x6e, 0x65, 0x78, 0x75,
	0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e,
	0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xd3, 0x01, 0x0a, 0x06,
	0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x42, 0x0a, 0x09, 0x73, 0x75, 0x63, 0x63, 0x65, 0x65,
	0x64, 0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x6e, 0x65, 0x78, 0x75,
	0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74,
	0x2e, 0x76, 0x31, 0x2e, 0x53, 0x75, 0x63, 0x63, 0x65, 0x65, 0x64, 0x65, 0x64, 0x48, 0x00, 0x52,
	0x09, 0x73, 0x75, 0x63, 0x63, 0x65, 0x65, 0x64, 0x65, 0x64, 0x12, 0x39, 0x0a, 0x06, 0x66, 0x61,
	0x69, 0x6c, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6e, 0x65, 0x78,
	0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c,
	0x74, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x48, 0x00, 0x52, 0x06, 0x66,
	0x61, 0x69, 0x6c, 0x65, 0x64, 0x12, 0x3f, 0x0a, 0x08, 0x63, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x65,
	0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e,
	0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x76,
	0x31, 0x2e, 0x43, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x65, 0x64, 0x48, 0x00, 0x52, 0x08, 0x63, 0x61,
	0x6e, 0x63, 0x65, 0x6c, 0x65, 0x64, 0x42, 0x09, 0x0a, 0x07, 0x76, 0x61, 0x72, 0x69, 0x61, 0x6e,
	0x74, 0x42, 0x5f, 0x0a, 0x17, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x76, 0x31, 0x42, 0x0b, 0x52, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x35, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2d, 0x72, 0x70,
	0x63, 0x2f, 0x73, 0x64, 0x6b, 0x2d, 0x67, 0x6f, 0x2f, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x61, 0x70,
	0x69, 0x2f, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2f, 0x76, 0x31, 0x3b, 0x72, 0x65, 0x73, 0x75,
	0x6c, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_nexus_rpc_api_result_v1_result_proto_rawDescOnce sync.Once
	file_nexus_rpc_api_result_v1_result_proto_rawDescData = file_nexus_rpc_api_result_v1_result_proto_rawDesc
)

func file_nexus_rpc_api_result_v1_result_proto_rawDescGZIP() []byte {
	file_nexus_rpc_api_result_v1_result_proto_rawDescOnce.Do(func() {
		file_nexus_rpc_api_result_v1_result_proto_rawDescData = protoimpl.X.CompressGZIP(file_nexus_rpc_api_result_v1_result_proto_rawDescData)
	})
	return file_nexus_rpc_api_result_v1_result_proto_rawDescData
}

var file_nexus_rpc_api_result_v1_result_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_nexus_rpc_api_result_v1_result_proto_goTypes = []interface{}{
	(*Succeeded)(nil),       // 0: nexus.rpc.api.result.v1.Succeeded
	(*Failed)(nil),          // 1: nexus.rpc.api.result.v1.Failed
	(*Canceled)(nil),        // 2: nexus.rpc.api.result.v1.Canceled
	(*Result)(nil),          // 3: nexus.rpc.api.result.v1.Result
	nil,                     // 4: nexus.rpc.api.result.v1.Succeeded.HeadersEntry
	nil,                     // 5: nexus.rpc.api.result.v1.Failed.HeadersEntry
	nil,                     // 6: nexus.rpc.api.result.v1.Canceled.HeadersEntry
	(*v1.HeaderValues)(nil), // 7: nexus.rpc.api.common.v1.HeaderValues
}
var file_nexus_rpc_api_result_v1_result_proto_depIdxs = []int32{
	4, // 0: nexus.rpc.api.result.v1.Succeeded.headers:type_name -> nexus.rpc.api.result.v1.Succeeded.HeadersEntry
	5, // 1: nexus.rpc.api.result.v1.Failed.headers:type_name -> nexus.rpc.api.result.v1.Failed.HeadersEntry
	6, // 2: nexus.rpc.api.result.v1.Canceled.headers:type_name -> nexus.rpc.api.result.v1.Canceled.HeadersEntry
	0, // 3: nexus.rpc.api.result.v1.Result.succeeded:type_name -> nexus.rpc.api.result.v1.Succeeded
	1, // 4: nexus.rpc.api.result.v1.Result.failed:type_name -> nexus.rpc.api.result.v1.Failed
	2, // 5: nexus.rpc.api.result.v1.Result.canceled:type_name -> nexus.rpc.api.result.v1.Canceled
	7, // 6: nexus.rpc.api.result.v1.Succeeded.HeadersEntry.value:type_name -> nexus.rpc.api.common.v1.HeaderValues
	7, // 7: nexus.rpc.api.result.v1.Failed.HeadersEntry.value:type_name -> nexus.rpc.api.common.v1.HeaderValues
	7, // 8: nexus.rpc.api.result.v1.Canceled.HeadersEntry.value:type_name -> nexus.rpc.api.common.v1.HeaderValues
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_nexus_rpc_api_result_v1_result_proto_init() }
func file_nexus_rpc_api_result_v1_result_proto_init() {
	if File_nexus_rpc_api_result_v1_result_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_nexus_rpc_api_result_v1_result_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Succeeded); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_nexus_rpc_api_result_v1_result_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Failed); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_nexus_rpc_api_result_v1_result_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Canceled); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_nexus_rpc_api_result_v1_result_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Result); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_nexus_rpc_api_result_v1_result_proto_msgTypes[3].OneofWrappers = []interface{}{
		(*Result_Succeeded)(nil),
		(*Result_Failed)(nil),
		(*Result_Canceled)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_nexus_rpc_api_result_v1_result_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_nexus_rpc_api_result_v1_result_proto_goTypes,
		DependencyIndexes: file_nexus_rpc_api_result_v1_result_proto_depIdxs,
		MessageInfos:      file_nexus_rpc_api_result_v1_result_proto_msgTypes,
	}.Build()
	File_nexus_rpc_api_result_v1_result_proto = out.File
	file_nexus_rpc_api_result_v1_result_proto_rawDesc = nil
	file_nexus_rpc_api_result_v1_result_proto_goTypes = nil
	file_nexus_rpc_api_result_v1_result_proto_depIdxs = nil
}
