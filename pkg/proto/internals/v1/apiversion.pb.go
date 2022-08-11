//
//Copyright 2021 The Dapr Authors
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//http://www.apache.org/licenses/LICENSE-2.0
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.1
// source: dapr/proto/internals/v1/apiversion.proto

package internals

import (
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

// APIVersion represents the version of Dapr Runtime API.
type APIVersion int32

const (
	// unspecified apiversion
	APIVersion_APIVERSION_UNSPECIFIED APIVersion = 0
	// Dapr API v1
	APIVersion_V1 APIVersion = 1
)

// Enum value maps for APIVersion.
var (
	APIVersion_name = map[int32]string{
		0: "APIVERSION_UNSPECIFIED",
		1: "V1",
	}
	APIVersion_value = map[string]int32{
		"APIVERSION_UNSPECIFIED": 0,
		"V1":                     1,
	}
)

func (x APIVersion) Enum() *APIVersion {
	p := new(APIVersion)
	*p = x
	return p
}

func (x APIVersion) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (APIVersion) Descriptor() protoreflect.EnumDescriptor {
	return file_dapr_proto_internals_v1_apiversion_proto_enumTypes[0].Descriptor()
}

func (APIVersion) Type() protoreflect.EnumType {
	return &file_dapr_proto_internals_v1_apiversion_proto_enumTypes[0]
}

func (x APIVersion) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use APIVersion.Descriptor instead.
func (APIVersion) EnumDescriptor() ([]byte, []int) {
	return file_dapr_proto_internals_v1_apiversion_proto_rawDescGZIP(), []int{0}
}

var File_dapr_proto_internals_v1_apiversion_proto protoreflect.FileDescriptor

var file_dapr_proto_internals_v1_apiversion_proto_rawDesc = []byte{
	0x0a, 0x28, 0x64, 0x61, 0x70, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x70, 0x69, 0x76, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x17, 0x64, 0x61, 0x70, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x73,
	0x2e, 0x76, 0x31, 0x2a, 0x30, 0x0a, 0x0a, 0x41, 0x50, 0x49, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f,
	0x6e, 0x12, 0x1a, 0x0a, 0x16, 0x41, 0x50, 0x49, 0x56, 0x45, 0x52, 0x53, 0x49, 0x4f, 0x4e, 0x5f,
	0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x06, 0x0a,
	0x02, 0x56, 0x31, 0x10, 0x01, 0x42, 0x37, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x61, 0x70, 0x72, 0x2f, 0x64, 0x61, 0x70, 0x72, 0x2f, 0x70, 0x6b,
	0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c,
	0x73, 0x2f, 0x76, 0x31, 0x3b, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x73, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_dapr_proto_internals_v1_apiversion_proto_rawDescOnce sync.Once
	file_dapr_proto_internals_v1_apiversion_proto_rawDescData = file_dapr_proto_internals_v1_apiversion_proto_rawDesc
)

func file_dapr_proto_internals_v1_apiversion_proto_rawDescGZIP() []byte {
	file_dapr_proto_internals_v1_apiversion_proto_rawDescOnce.Do(func() {
		file_dapr_proto_internals_v1_apiversion_proto_rawDescData = protoimpl.X.CompressGZIP(file_dapr_proto_internals_v1_apiversion_proto_rawDescData)
	})
	return file_dapr_proto_internals_v1_apiversion_proto_rawDescData
}

var file_dapr_proto_internals_v1_apiversion_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_dapr_proto_internals_v1_apiversion_proto_goTypes = []interface{}{
	(APIVersion)(0), // 0: dapr.proto.internals.v1.APIVersion
}
var file_dapr_proto_internals_v1_apiversion_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_dapr_proto_internals_v1_apiversion_proto_init() }
func file_dapr_proto_internals_v1_apiversion_proto_init() {
	if File_dapr_proto_internals_v1_apiversion_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_dapr_proto_internals_v1_apiversion_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_dapr_proto_internals_v1_apiversion_proto_goTypes,
		DependencyIndexes: file_dapr_proto_internals_v1_apiversion_proto_depIdxs,
		EnumInfos:         file_dapr_proto_internals_v1_apiversion_proto_enumTypes,
	}.Build()
	File_dapr_proto_internals_v1_apiversion_proto = out.File
	file_dapr_proto_internals_v1_apiversion_proto_rawDesc = nil
	file_dapr_proto_internals_v1_apiversion_proto_goTypes = nil
	file_dapr_proto_internals_v1_apiversion_proto_depIdxs = nil
}
