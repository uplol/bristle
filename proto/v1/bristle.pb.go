// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.6.1
// source: bristle.proto

package v1

import (
	context "context"
	proto "github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

// A payload containing multiple proto bodies for the given descriptor type
type Payload struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type string   `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	Body [][]byte `protobuf:"bytes,2,rep,name=body,proto3" json:"body,omitempty"`
}

func (x *Payload) Reset() {
	*x = Payload{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bristle_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Payload) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Payload) ProtoMessage() {}

func (x *Payload) ProtoReflect() protoreflect.Message {
	mi := &file_bristle_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Payload.ProtoReflect.Descriptor instead.
func (*Payload) Descriptor() ([]byte, []int) {
	return file_bristle_proto_rawDescGZIP(), []int{0}
}

func (x *Payload) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *Payload) GetBody() [][]byte {
	if x != nil {
		return x.Body
	}
	return nil
}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bristle_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_bristle_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_bristle_proto_rawDescGZIP(), []int{1}
}

type WriteBatchRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key      string     `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Payloads []*Payload `protobuf:"bytes,2,rep,name=payloads,proto3" json:"payloads,omitempty"`
}

func (x *WriteBatchRequest) Reset() {
	*x = WriteBatchRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bristle_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WriteBatchRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WriteBatchRequest) ProtoMessage() {}

func (x *WriteBatchRequest) ProtoReflect() protoreflect.Message {
	mi := &file_bristle_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WriteBatchRequest.ProtoReflect.Descriptor instead.
func (*WriteBatchRequest) Descriptor() ([]byte, []int) {
	return file_bristle_proto_rawDescGZIP(), []int{2}
}

func (x *WriteBatchRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *WriteBatchRequest) GetPayloads() []*Payload {
	if x != nil {
		return x.Payloads
	}
	return nil
}

type WriteBatchResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The number of payloads that where acknowledged
	Acknowledged uint64 `protobuf:"varint,1,opt,name=acknowledged,proto3" json:"acknowledged,omitempty"`
	// The number of payloads that where dropped
	Dropped uint64 `protobuf:"varint,2,opt,name=dropped,proto3" json:"dropped,omitempty"`
}

func (x *WriteBatchResponse) Reset() {
	*x = WriteBatchResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_bristle_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WriteBatchResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WriteBatchResponse) ProtoMessage() {}

func (x *WriteBatchResponse) ProtoReflect() protoreflect.Message {
	mi := &file_bristle_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WriteBatchResponse.ProtoReflect.Descriptor instead.
func (*WriteBatchResponse) Descriptor() ([]byte, []int) {
	return file_bristle_proto_rawDescGZIP(), []int{3}
}

func (x *WriteBatchResponse) GetAcknowledged() uint64 {
	if x != nil {
		return x.Acknowledged
	}
	return 0
}

func (x *WriteBatchResponse) GetDropped() uint64 {
	if x != nil {
		return x.Dropped
	}
	return 0
}

var file_bristle_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptor.MessageOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         50001,
		Name:          "bristle.bristle_table",
		Tag:           "bytes,50001,opt,name=bristle_table",
		Filename:      "bristle.proto",
	},
	{
		ExtendedType:  (*descriptor.FieldOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         50001,
		Name:          "bristle.bristle_column",
		Tag:           "bytes,50001,opt,name=bristle_column",
		Filename:      "bristle.proto",
	},
	{
		ExtendedType:  (*descriptor.FieldOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         50002,
		Name:          "bristle.bristle_clickhouse_type",
		Tag:           "bytes,50002,opt,name=bristle_clickhouse_type",
		Filename:      "bristle.proto",
	},
}

// Extension fields to descriptor.MessageOptions.
var (
	// optional string bristle_table = 50001;
	E_BristleTable = &file_bristle_proto_extTypes[0]
)

// Extension fields to descriptor.FieldOptions.
var (
	// optional string bristle_column = 50001;
	E_BristleColumn = &file_bristle_proto_extTypes[1]
	// optional string bristle_clickhouse_type = 50002;
	E_BristleClickhouseType = &file_bristle_proto_extTypes[2]
)

var File_bristle_proto protoreflect.FileDescriptor

var file_bristle_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x07, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x31, 0x0a, 0x07, 0x50, 0x61,
	0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6f, 0x64,
	0x79, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x22, 0x07, 0x0a,
	0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x53, 0x0a, 0x11, 0x57, 0x72, 0x69, 0x74, 0x65, 0x42,
	0x61, 0x74, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x2c, 0x0a,
	0x08, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x10, 0x2e, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x2e, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61,
	0x64, 0x52, 0x08, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x73, 0x22, 0x52, 0x0a, 0x12, 0x57,
	0x72, 0x69, 0x74, 0x65, 0x42, 0x61, 0x74, 0x63, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x22, 0x0a, 0x0c, 0x61, 0x63, 0x6b, 0x6e, 0x6f, 0x77, 0x6c, 0x65, 0x64, 0x67, 0x65,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0c, 0x61, 0x63, 0x6b, 0x6e, 0x6f, 0x77, 0x6c,
	0x65, 0x64, 0x67, 0x65, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x64, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x64, 0x72, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x32,
	0xa2, 0x01, 0x0a, 0x14, 0x42, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x49, 0x6e, 0x67, 0x65, 0x73,
	0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x45, 0x0a, 0x0a, 0x57, 0x72, 0x69, 0x74,
	0x65, 0x42, 0x61, 0x74, 0x63, 0x68, 0x12, 0x1a, 0x2e, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65,
	0x2e, 0x57, 0x72, 0x69, 0x74, 0x65, 0x42, 0x61, 0x74, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x2e, 0x57, 0x72, 0x69,
	0x74, 0x65, 0x42, 0x61, 0x74, 0x63, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x43, 0x0a, 0x13, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x69, 0x6e, 0x67, 0x57, 0x72, 0x69, 0x74,
	0x65, 0x42, 0x61, 0x74, 0x63, 0x68, 0x12, 0x1a, 0x2e, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65,
	0x2e, 0x57, 0x72, 0x69, 0x74, 0x65, 0x42, 0x61, 0x74, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x0e, 0x2e, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x28, 0x01, 0x3a, 0x46, 0x0a, 0x0d, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x5f,
	0x74, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x1f, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xd1, 0x86, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c,
	0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x3a, 0x46, 0x0a, 0x0e,
	0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x5f, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x12, 0x1d,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xd1, 0x86,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x43, 0x6f,
	0x6c, 0x75, 0x6d, 0x6e, 0x3a, 0x57, 0x0a, 0x17, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x5f,
	0x63, 0x6c, 0x69, 0x63, 0x6b, 0x68, 0x6f, 0x75, 0x73, 0x65, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x12,
	0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xd2,
	0x86, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x15, 0x62, 0x72, 0x69, 0x73, 0x74, 0x6c, 0x65, 0x43,
	0x6c, 0x69, 0x63, 0x6b, 0x68, 0x6f, 0x75, 0x73, 0x65, 0x54, 0x79, 0x70, 0x65, 0x42, 0x0a, 0x5a,
	0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_bristle_proto_rawDescOnce sync.Once
	file_bristle_proto_rawDescData = file_bristle_proto_rawDesc
)

func file_bristle_proto_rawDescGZIP() []byte {
	file_bristle_proto_rawDescOnce.Do(func() {
		file_bristle_proto_rawDescData = protoimpl.X.CompressGZIP(file_bristle_proto_rawDescData)
	})
	return file_bristle_proto_rawDescData
}

var file_bristle_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_bristle_proto_goTypes = []interface{}{
	(*Payload)(nil),                   // 0: bristle.Payload
	(*Empty)(nil),                     // 1: bristle.Empty
	(*WriteBatchRequest)(nil),         // 2: bristle.WriteBatchRequest
	(*WriteBatchResponse)(nil),        // 3: bristle.WriteBatchResponse
	(*descriptor.MessageOptions)(nil), // 4: google.protobuf.MessageOptions
	(*descriptor.FieldOptions)(nil),   // 5: google.protobuf.FieldOptions
}
var file_bristle_proto_depIdxs = []int32{
	0, // 0: bristle.WriteBatchRequest.payloads:type_name -> bristle.Payload
	4, // 1: bristle.bristle_table:extendee -> google.protobuf.MessageOptions
	5, // 2: bristle.bristle_column:extendee -> google.protobuf.FieldOptions
	5, // 3: bristle.bristle_clickhouse_type:extendee -> google.protobuf.FieldOptions
	2, // 4: bristle.BristleIngestService.WriteBatch:input_type -> bristle.WriteBatchRequest
	2, // 5: bristle.BristleIngestService.StreamingWriteBatch:input_type -> bristle.WriteBatchRequest
	3, // 6: bristle.BristleIngestService.WriteBatch:output_type -> bristle.WriteBatchResponse
	1, // 7: bristle.BristleIngestService.StreamingWriteBatch:output_type -> bristle.Empty
	6, // [6:8] is the sub-list for method output_type
	4, // [4:6] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	1, // [1:4] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_bristle_proto_init() }
func file_bristle_proto_init() {
	if File_bristle_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_bristle_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Payload); i {
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
		file_bristle_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
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
		file_bristle_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WriteBatchRequest); i {
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
		file_bristle_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WriteBatchResponse); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_bristle_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 3,
			NumServices:   1,
		},
		GoTypes:           file_bristle_proto_goTypes,
		DependencyIndexes: file_bristle_proto_depIdxs,
		MessageInfos:      file_bristle_proto_msgTypes,
		ExtensionInfos:    file_bristle_proto_extTypes,
	}.Build()
	File_bristle_proto = out.File
	file_bristle_proto_rawDesc = nil
	file_bristle_proto_goTypes = nil
	file_bristle_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// BristleIngestServiceClient is the client API for BristleIngestService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BristleIngestServiceClient interface {
	// Writes a single batch containing multiple payloads
	WriteBatch(ctx context.Context, in *WriteBatchRequest, opts ...grpc.CallOption) (*WriteBatchResponse, error)
	// Streaming write of batches containing multiple payloads
	StreamingWriteBatch(ctx context.Context, opts ...grpc.CallOption) (BristleIngestService_StreamingWriteBatchClient, error)
}

type bristleIngestServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBristleIngestServiceClient(cc grpc.ClientConnInterface) BristleIngestServiceClient {
	return &bristleIngestServiceClient{cc}
}

func (c *bristleIngestServiceClient) WriteBatch(ctx context.Context, in *WriteBatchRequest, opts ...grpc.CallOption) (*WriteBatchResponse, error) {
	out := new(WriteBatchResponse)
	err := c.cc.Invoke(ctx, "/bristle.BristleIngestService/WriteBatch", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bristleIngestServiceClient) StreamingWriteBatch(ctx context.Context, opts ...grpc.CallOption) (BristleIngestService_StreamingWriteBatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &_BristleIngestService_serviceDesc.Streams[0], "/bristle.BristleIngestService/StreamingWriteBatch", opts...)
	if err != nil {
		return nil, err
	}
	x := &bristleIngestServiceStreamingWriteBatchClient{stream}
	return x, nil
}

type BristleIngestService_StreamingWriteBatchClient interface {
	Send(*WriteBatchRequest) error
	CloseAndRecv() (*Empty, error)
	grpc.ClientStream
}

type bristleIngestServiceStreamingWriteBatchClient struct {
	grpc.ClientStream
}

func (x *bristleIngestServiceStreamingWriteBatchClient) Send(m *WriteBatchRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *bristleIngestServiceStreamingWriteBatchClient) CloseAndRecv() (*Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// BristleIngestServiceServer is the server API for BristleIngestService service.
type BristleIngestServiceServer interface {
	// Writes a single batch containing multiple payloads
	WriteBatch(context.Context, *WriteBatchRequest) (*WriteBatchResponse, error)
	// Streaming write of batches containing multiple payloads
	StreamingWriteBatch(BristleIngestService_StreamingWriteBatchServer) error
}

// UnimplementedBristleIngestServiceServer can be embedded to have forward compatible implementations.
type UnimplementedBristleIngestServiceServer struct {
}

func (*UnimplementedBristleIngestServiceServer) WriteBatch(context.Context, *WriteBatchRequest) (*WriteBatchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WriteBatch not implemented")
}
func (*UnimplementedBristleIngestServiceServer) StreamingWriteBatch(BristleIngestService_StreamingWriteBatchServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamingWriteBatch not implemented")
}

func RegisterBristleIngestServiceServer(s *grpc.Server, srv BristleIngestServiceServer) {
	s.RegisterService(&_BristleIngestService_serviceDesc, srv)
}

func _BristleIngestService_WriteBatch_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteBatchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BristleIngestServiceServer).WriteBatch(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bristle.BristleIngestService/WriteBatch",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BristleIngestServiceServer).WriteBatch(ctx, req.(*WriteBatchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BristleIngestService_StreamingWriteBatch_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BristleIngestServiceServer).StreamingWriteBatch(&bristleIngestServiceStreamingWriteBatchServer{stream})
}

type BristleIngestService_StreamingWriteBatchServer interface {
	SendAndClose(*Empty) error
	Recv() (*WriteBatchRequest, error)
	grpc.ServerStream
}

type bristleIngestServiceStreamingWriteBatchServer struct {
	grpc.ServerStream
}

func (x *bristleIngestServiceStreamingWriteBatchServer) SendAndClose(m *Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *bristleIngestServiceStreamingWriteBatchServer) Recv() (*WriteBatchRequest, error) {
	m := new(WriteBatchRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _BristleIngestService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "bristle.BristleIngestService",
	HandlerType: (*BristleIngestServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "WriteBatch",
			Handler:    _BristleIngestService_WriteBatch_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamingWriteBatch",
			Handler:       _BristleIngestService_StreamingWriteBatch_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "bristle.proto",
}