// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.12
// source: nexus/rpc/api/nexusservice/v1/service.proto

package nexusservice

import (
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_nexus_rpc_api_nexusservice_v1_service_proto protoreflect.FileDescriptor

var file_nexus_rpc_api_nexusservice_v1_service_proto_rawDesc = []byte{
	0x0a, 0x2b, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x6e,
	0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x6e, 0x65, 0x78,
	0x75, 0x73, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x1a, 0x1c, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x34, 0x6e, 0x65, 0x78, 0x75,
	0x73, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x5f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x32, 0xcc, 0x01, 0x0a, 0x0c, 0x4e, 0x65, 0x78, 0x75, 0x73, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x12, 0xbb, 0x01, 0x0a, 0x0e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x4f, 0x70, 0x65, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x34, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x35, 0x2e, 0x6e, 0x65, 0x78,
	0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x72, 0x74,
	0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x3c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x36, 0x3a, 0x01, 0x2a, 0x22, 0x31, 0x2f, 0x76,
	0x31, 0x2f, 0x7b, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x3d, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x73, 0x2f, 0x2a, 0x7d, 0x2f, 0x7b, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x3d, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x2a, 0x7d, 0x42,
	0x72, 0x0a, 0x1d, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x76, 0x31,
	0x42, 0x0c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01,
	0x5a, 0x41, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6e, 0x65, 0x78,
	0x75, 0x73, 0x2d, 0x72, 0x70, 0x63, 0x2f, 0x73, 0x64, 0x6b, 0x2d, 0x67, 0x6f, 0x2f, 0x6e, 0x65,
	0x78, 0x75, 0x73, 0x61, 0x70, 0x69, 0x2f, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x3b, 0x6e, 0x65, 0x78, 0x75, 0x73, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_nexus_rpc_api_nexusservice_v1_service_proto_goTypes = []interface{}{
	(*StartOperationRequest)(nil),  // 0: nexus.rpc.api.nexusservice.v1.StartOperationRequest
	(*StartOperationResponse)(nil), // 1: nexus.rpc.api.nexusservice.v1.StartOperationResponse
}
var file_nexus_rpc_api_nexusservice_v1_service_proto_depIdxs = []int32{
	0, // 0: nexus.rpc.api.nexusservice.v1.NexusService.StartOperation:input_type -> nexus.rpc.api.nexusservice.v1.StartOperationRequest
	1, // 1: nexus.rpc.api.nexusservice.v1.NexusService.StartOperation:output_type -> nexus.rpc.api.nexusservice.v1.StartOperationResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_nexus_rpc_api_nexusservice_v1_service_proto_init() }
func file_nexus_rpc_api_nexusservice_v1_service_proto_init() {
	if File_nexus_rpc_api_nexusservice_v1_service_proto != nil {
		return
	}
	file_nexus_rpc_api_nexusservice_v1_request_response_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_nexus_rpc_api_nexusservice_v1_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_nexus_rpc_api_nexusservice_v1_service_proto_goTypes,
		DependencyIndexes: file_nexus_rpc_api_nexusservice_v1_service_proto_depIdxs,
	}.Build()
	File_nexus_rpc_api_nexusservice_v1_service_proto = out.File
	file_nexus_rpc_api_nexusservice_v1_service_proto_rawDesc = nil
	file_nexus_rpc_api_nexusservice_v1_service_proto_goTypes = nil
	file_nexus_rpc_api_nexusservice_v1_service_proto_depIdxs = nil
}
