// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.12
// source: nexus/rpc/api/nexusservice/v1/request_response.proto

package nexusservice

import (
	v1 "github.com/nexus-rpc/sdk-go/nexusapi/common/v1"
	v11 "github.com/nexus-rpc/sdk-go/nexusapi/result/v1"
	_ "google.golang.org/genproto/googleapis/api/annotations"
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

// Start an operation by type and input.
type StartOperationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the service of this operation.
	// When exposed over HTTP, service is be part of the URL.
	// When exposed over gRPC, service will be part of the request body. It should be noted that a proxy would need to
	// read and decode the request body in order to extract the service, which can potentially be an expensive operation.
	Service string `protobuf:"bytes,1,opt,name=service,proto3" json:"service,omitempty"`
	// Name of the operation.
	Operation string                      `protobuf:"bytes,2,opt,name=operation,proto3" json:"operation,omitempty"`
	Headers   map[string]*v1.HeaderValues `protobuf:"bytes,3,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Optional identifier that may be used to to reference this operation, e.g. to cancel it.
	// If not provided, and the operation completes asynchronously, the handler will allocate an ID in the response.
	OperationId string `protobuf:"bytes,4,opt,name=operation_id,json=operationId,proto3" json:"operation_id,omitempty"`
	// Opaque request input.
	// The content type and encoding of this input may be provided via request headers.
	Body []byte `protobuf:"bytes,5,opt,name=body,proto3" json:"body,omitempty"`
	// Optional callback URL.
	// If this operation is asynchronous and the handler supports callbacks, the handler should call this URL when the
	// operation's result (succeeded, failed, canceled) becomes available.
	CallbackUri string `protobuf:"bytes,6,opt,name=callback_uri,json=callbackUri,proto3" json:"callback_uri,omitempty"`
}

func (x *StartOperationRequest) Reset() {
	*x = StartOperationRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartOperationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartOperationRequest) ProtoMessage() {}

func (x *StartOperationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartOperationRequest.ProtoReflect.Descriptor instead.
func (*StartOperationRequest) Descriptor() ([]byte, []int) {
	return file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescGZIP(), []int{0}
}

func (x *StartOperationRequest) GetService() string {
	if x != nil {
		return x.Service
	}
	return ""
}

func (x *StartOperationRequest) GetOperation() string {
	if x != nil {
		return x.Operation
	}
	return ""
}

func (x *StartOperationRequest) GetHeaders() map[string]*v1.HeaderValues {
	if x != nil {
		return x.Headers
	}
	return nil
}

func (x *StartOperationRequest) GetOperationId() string {
	if x != nil {
		return x.OperationId
	}
	return ""
}

func (x *StartOperationRequest) GetBody() []byte {
	if x != nil {
		return x.Body
	}
	return nil
}

func (x *StartOperationRequest) GetCallbackUri() string {
	if x != nil {
		return x.CallbackUri
	}
	return ""
}

type StartOperationResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Result:
	//
	//	*StartOperationResponse_Started_
	//	*StartOperationResponse_Succeeded
	//	*StartOperationResponse_Failed
	//	*StartOperationResponse_Canceled
	Result isStartOperationResponse_Result `protobuf_oneof:"result"`
}

func (x *StartOperationResponse) Reset() {
	*x = StartOperationResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartOperationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartOperationResponse) ProtoMessage() {}

func (x *StartOperationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartOperationResponse.ProtoReflect.Descriptor instead.
func (*StartOperationResponse) Descriptor() ([]byte, []int) {
	return file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescGZIP(), []int{1}
}

func (m *StartOperationResponse) GetResult() isStartOperationResponse_Result {
	if m != nil {
		return m.Result
	}
	return nil
}

func (x *StartOperationResponse) GetStarted() *StartOperationResponse_Started {
	if x, ok := x.GetResult().(*StartOperationResponse_Started_); ok {
		return x.Started
	}
	return nil
}

func (x *StartOperationResponse) GetSucceeded() *v11.Succeeded {
	if x, ok := x.GetResult().(*StartOperationResponse_Succeeded); ok {
		return x.Succeeded
	}
	return nil
}

func (x *StartOperationResponse) GetFailed() *v11.Failed {
	if x, ok := x.GetResult().(*StartOperationResponse_Failed); ok {
		return x.Failed
	}
	return nil
}

func (x *StartOperationResponse) GetCanceled() *v11.Canceled {
	if x, ok := x.GetResult().(*StartOperationResponse_Canceled); ok {
		return x.Canceled
	}
	return nil
}

type isStartOperationResponse_Result interface {
	isStartOperationResponse_Result()
}

type StartOperationResponse_Started_ struct {
	Started *StartOperationResponse_Started `protobuf:"bytes,1,opt,name=started,proto3,oneof"`
}

type StartOperationResponse_Succeeded struct {
	Succeeded *v11.Succeeded `protobuf:"bytes,2,opt,name=succeeded,proto3,oneof"`
}

type StartOperationResponse_Failed struct {
	Failed *v11.Failed `protobuf:"bytes,3,opt,name=failed,proto3,oneof"`
}

type StartOperationResponse_Canceled struct {
	Canceled *v11.Canceled `protobuf:"bytes,4,opt,name=canceled,proto3,oneof"`
}

func (*StartOperationResponse_Started_) isStartOperationResponse_Result() {}

func (*StartOperationResponse_Succeeded) isStartOperationResponse_Result() {}

func (*StartOperationResponse_Failed) isStartOperationResponse_Result() {}

func (*StartOperationResponse_Canceled) isStartOperationResponse_Result() {}

// The operation has been started and will complete asynchronously.
// Use the other NexusService methods to cancel it or get its status or result.
type StartOperationResponse_Started struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Headers map[string]*v1.HeaderValues `protobuf:"bytes,1,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Identifier that may be used to to reference this operation, e.g. to cancel it.
	OperationId string `protobuf:"bytes,2,opt,name=operation_id,json=operationId,proto3" json:"operation_id,omitempty"`
	// If the request specified a callback_uri and the handler supports callbacks for this operation, this flag will
	// be set.
	// It is up to the caller to decided how to get the outcome of this call in case the handler does not support
	// callbacks.
	CallbackUriSupported bool `protobuf:"varint,3,opt,name=callback_uri_supported,json=callbackUriSupported,proto3" json:"callback_uri_supported,omitempty"`
}

func (x *StartOperationResponse_Started) Reset() {
	*x = StartOperationResponse_Started{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartOperationResponse_Started) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartOperationResponse_Started) ProtoMessage() {}

func (x *StartOperationResponse_Started) ProtoReflect() protoreflect.Message {
	mi := &file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartOperationResponse_Started.ProtoReflect.Descriptor instead.
func (*StartOperationResponse_Started) Descriptor() ([]byte, []int) {
	return file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescGZIP(), []int{1, 0}
}

func (x *StartOperationResponse_Started) GetHeaders() map[string]*v1.HeaderValues {
	if x != nil {
		return x.Headers
	}
	return nil
}

func (x *StartOperationResponse_Started) GetOperationId() string {
	if x != nil {
		return x.OperationId
	}
	return ""
}

func (x *StartOperationResponse_Started) GetCallbackUriSupported() bool {
	if x != nil {
		return x.CallbackUriSupported
	}
	return false
}

var File_nexus_rpc_api_nexusservice_v1_request_response_proto protoreflect.FileDescriptor

var file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDesc = []byte{
	0x0a, 0x34, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f,
	0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70,
	0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x5f, 0x62, 0x65, 0x68, 0x61, 0x76, 0x69, 0x6f, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x24, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2f, 0x72, 0x70,
	0x63, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2f, 0x76, 0x31, 0x2f,
	0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x24, 0x6e, 0x65,
	0x78, 0x75, 0x73, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x72, 0x65, 0x73, 0x75,
	0x6c, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x87, 0x03, 0x0a, 0x15, 0x53, 0x74, 0x61, 0x72, 0x74, 0x4f, 0x70, 0x65, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x07,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x03, 0xe0,
	0x41, 0x02, 0x52, 0x07, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x21, 0x0a, 0x09, 0x6f,
	0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x03,
	0xe0, 0x41, 0x02, 0x52, 0x09, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x60,
	0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x41, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e,
	0x53, 0x74, 0x61, 0x72, 0x74, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x42, 0x03, 0xe0, 0x41, 0x01, 0x52, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73,
	0x12, 0x26, 0x0a, 0x0c, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x42, 0x03, 0xe0, 0x41, 0x01, 0x52, 0x0b, 0x6f, 0x70, 0x65,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x04, 0x62, 0x6f, 0x64, 0x79,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x42, 0x03, 0xe0, 0x41, 0x01, 0x52, 0x04, 0x62, 0x6f, 0x64,
	0x79, 0x12, 0x26, 0x0a, 0x0c, 0x63, 0x61, 0x6c, 0x6c, 0x62, 0x61, 0x63, 0x6b, 0x5f, 0x75, 0x72,
	0x69, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x42, 0x03, 0xe0, 0x41, 0x01, 0x52, 0x0b, 0x63, 0x61,
	0x6c, 0x6c, 0x62, 0x61, 0x63, 0x6b, 0x55, 0x72, 0x69, 0x1a, 0x61, 0x0a, 0x0c, 0x48, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x3b, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x6e, 0x65, 0x78,
	0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f,
	0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x73, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xeb, 0x04, 0x0a,
	0x16, 0x53, 0x74, 0x61, 0x72, 0x74, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x59, 0x0a, 0x07, 0x73, 0x74, 0x61, 0x72, 0x74,
	0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x3d, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73,
	0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x4f, 0x70,
	0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e,
	0x53, 0x74, 0x61, 0x72, 0x74, 0x65, 0x64, 0x48, 0x00, 0x52, 0x07, 0x73, 0x74, 0x61, 0x72, 0x74,
	0x65, 0x64, 0x12, 0x42, 0x0a, 0x09, 0x73, 0x75, 0x63, 0x63, 0x65, 0x65, 0x64, 0x65, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70,
	0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x76, 0x31, 0x2e,
	0x53, 0x75, 0x63, 0x63, 0x65, 0x65, 0x64, 0x65, 0x64, 0x48, 0x00, 0x52, 0x09, 0x73, 0x75, 0x63,
	0x63, 0x65, 0x65, 0x64, 0x65, 0x64, 0x12, 0x39, 0x0a, 0x06, 0x66, 0x61, 0x69, 0x6c, 0x65, 0x64,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72,
	0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x76, 0x31,
	0x2e, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x48, 0x00, 0x52, 0x06, 0x66, 0x61, 0x69, 0x6c, 0x65,
	0x64, 0x12, 0x3f, 0x0a, 0x08, 0x63, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x65, 0x64, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x61,
	0x6e, 0x63, 0x65, 0x6c, 0x65, 0x64, 0x48, 0x00, 0x52, 0x08, 0x63, 0x61, 0x6e, 0x63, 0x65, 0x6c,
	0x65, 0x64, 0x1a, 0xab, 0x02, 0x0a, 0x07, 0x53, 0x74, 0x61, 0x72, 0x74, 0x65, 0x64, 0x12, 0x64,
	0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x4a, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e,
	0x53, 0x74, 0x61, 0x72, 0x74, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x65, 0x64, 0x2e, 0x48,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x68, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x73, 0x12, 0x21, 0x0a, 0x0c, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x6f, 0x70, 0x65, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x34, 0x0a, 0x16, 0x63, 0x61, 0x6c, 0x6c, 0x62,
	0x61, 0x63, 0x6b, 0x5f, 0x75, 0x72, 0x69, 0x5f, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x65,
	0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x14, 0x63, 0x61, 0x6c, 0x6c, 0x62, 0x61, 0x63,
	0x6b, 0x55, 0x72, 0x69, 0x53, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x64, 0x1a, 0x61, 0x0a,
	0x0c, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x3b, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25,
	0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x63,
	0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x42, 0x08, 0x0a, 0x06, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x42, 0x7a, 0x0a, 0x1d, 0x6e, 0x65,
	0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x6e, 0x65, 0x78, 0x75,
	0x73, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x42, 0x14, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x50, 0x72, 0x6f, 0x74,
	0x6f, 0x50, 0x01, 0x5a, 0x41, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x6e, 0x65, 0x78, 0x75, 0x73, 0x2d, 0x72, 0x70, 0x63, 0x2f, 0x73, 0x64, 0x6b, 0x2d, 0x67, 0x6f,
	0x2f, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x61, 0x70, 0x69, 0x2f, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x3b, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescOnce sync.Once
	file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescData = file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDesc
)

func file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescGZIP() []byte {
	file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescOnce.Do(func() {
		file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescData = protoimpl.X.CompressGZIP(file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescData)
	})
	return file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDescData
}

var file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_nexus_rpc_api_nexusservice_v1_request_response_proto_goTypes = []interface{}{
	(*StartOperationRequest)(nil),          // 0: nexus.rpc.api.nexusservice.v1.StartOperationRequest
	(*StartOperationResponse)(nil),         // 1: nexus.rpc.api.nexusservice.v1.StartOperationResponse
	nil,                                    // 2: nexus.rpc.api.nexusservice.v1.StartOperationRequest.HeadersEntry
	(*StartOperationResponse_Started)(nil), // 3: nexus.rpc.api.nexusservice.v1.StartOperationResponse.Started
	nil,                                    // 4: nexus.rpc.api.nexusservice.v1.StartOperationResponse.Started.HeadersEntry
	(*v11.Succeeded)(nil),                  // 5: nexus.rpc.api.result.v1.Succeeded
	(*v11.Failed)(nil),                     // 6: nexus.rpc.api.result.v1.Failed
	(*v11.Canceled)(nil),                   // 7: nexus.rpc.api.result.v1.Canceled
	(*v1.HeaderValues)(nil),                // 8: nexus.rpc.api.common.v1.HeaderValues
}
var file_nexus_rpc_api_nexusservice_v1_request_response_proto_depIdxs = []int32{
	2, // 0: nexus.rpc.api.nexusservice.v1.StartOperationRequest.headers:type_name -> nexus.rpc.api.nexusservice.v1.StartOperationRequest.HeadersEntry
	3, // 1: nexus.rpc.api.nexusservice.v1.StartOperationResponse.started:type_name -> nexus.rpc.api.nexusservice.v1.StartOperationResponse.Started
	5, // 2: nexus.rpc.api.nexusservice.v1.StartOperationResponse.succeeded:type_name -> nexus.rpc.api.result.v1.Succeeded
	6, // 3: nexus.rpc.api.nexusservice.v1.StartOperationResponse.failed:type_name -> nexus.rpc.api.result.v1.Failed
	7, // 4: nexus.rpc.api.nexusservice.v1.StartOperationResponse.canceled:type_name -> nexus.rpc.api.result.v1.Canceled
	8, // 5: nexus.rpc.api.nexusservice.v1.StartOperationRequest.HeadersEntry.value:type_name -> nexus.rpc.api.common.v1.HeaderValues
	4, // 6: nexus.rpc.api.nexusservice.v1.StartOperationResponse.Started.headers:type_name -> nexus.rpc.api.nexusservice.v1.StartOperationResponse.Started.HeadersEntry
	8, // 7: nexus.rpc.api.nexusservice.v1.StartOperationResponse.Started.HeadersEntry.value:type_name -> nexus.rpc.api.common.v1.HeaderValues
	8, // [8:8] is the sub-list for method output_type
	8, // [8:8] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_nexus_rpc_api_nexusservice_v1_request_response_proto_init() }
func file_nexus_rpc_api_nexusservice_v1_request_response_proto_init() {
	if File_nexus_rpc_api_nexusservice_v1_request_response_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartOperationRequest); i {
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
		file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartOperationResponse); i {
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
		file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartOperationResponse_Started); i {
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
	file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*StartOperationResponse_Started_)(nil),
		(*StartOperationResponse_Succeeded)(nil),
		(*StartOperationResponse_Failed)(nil),
		(*StartOperationResponse_Canceled)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_nexus_rpc_api_nexusservice_v1_request_response_proto_goTypes,
		DependencyIndexes: file_nexus_rpc_api_nexusservice_v1_request_response_proto_depIdxs,
		MessageInfos:      file_nexus_rpc_api_nexusservice_v1_request_response_proto_msgTypes,
	}.Build()
	File_nexus_rpc_api_nexusservice_v1_request_response_proto = out.File
	file_nexus_rpc_api_nexusservice_v1_request_response_proto_rawDesc = nil
	file_nexus_rpc_api_nexusservice_v1_request_response_proto_goTypes = nil
	file_nexus_rpc_api_nexusservice_v1_request_response_proto_depIdxs = nil
}
