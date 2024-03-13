// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.7.1
// source: msg.proto

package nppbmsg

import (
	proto "github.com/golang/protobuf/proto"
	md "github.com/huangjunwen/nproto/v2/pb/md"
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

// MessageWithMD
type MessageWithMD struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// MetaData key/values.
	MetaData map[string]*md.MetaDataValueList `protobuf:"bytes,1,rep,name=meta_data,json=metaData,proto3" json:"meta_data,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// MsgFormat is the wire format for MsgData.
	MsgFormat string `protobuf:"bytes,2,opt,name=msg_format,json=msgFormat,proto3" json:"msg_format,omitempty"`
	// MsgBytes is encoded msg data raw bytes.
	MsgBytes []byte `protobuf:"bytes,3,opt,name=msg_bytes,json=msgBytes,proto3" json:"msg_bytes,omitempty"`
}

func (x *MessageWithMD) Reset() {
	*x = MessageWithMD{}
	if protoimpl.UnsafeEnabled {
		mi := &file_msg_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MessageWithMD) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MessageWithMD) ProtoMessage() {}

func (x *MessageWithMD) ProtoReflect() protoreflect.Message {
	mi := &file_msg_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MessageWithMD.ProtoReflect.Descriptor instead.
func (*MessageWithMD) Descriptor() ([]byte, []int) {
	return file_msg_proto_rawDescGZIP(), []int{0}
}

func (x *MessageWithMD) GetMetaData() map[string]*md.MetaDataValueList {
	if x != nil {
		return x.MetaData
	}
	return nil
}

func (x *MessageWithMD) GetMsgFormat() string {
	if x != nil {
		return x.MsgFormat
	}
	return ""
}

func (x *MessageWithMD) GetMsgBytes() []byte {
	if x != nil {
		return x.MsgBytes
	}
	return nil
}

var File_msg_proto protoreflect.FileDescriptor

var file_msg_proto_rawDesc = []byte{
	0x0a, 0x09, 0x6d, 0x73, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0d, 0x6e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x70, 0x62, 0x2e, 0x6d, 0x73, 0x67, 0x1a, 0x0b, 0x6d, 0x64, 0x2f, 0x6d,
	0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xf2, 0x01, 0x0a, 0x0d, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x57, 0x69, 0x74, 0x68, 0x4d, 0x44, 0x12, 0x47, 0x0a, 0x09, 0x6d, 0x65, 0x74,
	0x61, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x6e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x70, 0x62, 0x2e, 0x6d, 0x73, 0x67, 0x2e, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x57, 0x69, 0x74, 0x68, 0x4d, 0x44, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x44,
	0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x44, 0x61,
	0x74, 0x61, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x73, 0x67, 0x5f, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6d, 0x73, 0x67, 0x46, 0x6f, 0x72, 0x6d, 0x61,
	0x74, 0x12, 0x1b, 0x0a, 0x09, 0x6d, 0x73, 0x67, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x08, 0x6d, 0x73, 0x67, 0x42, 0x79, 0x74, 0x65, 0x73, 0x1a, 0x5c,
	0x0a, 0x0d, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x35, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1f, 0x2e, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x70, 0x62, 0x2e, 0x6d, 0x64, 0x2e,
	0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x4c, 0x69, 0x73,
	0x74, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x31, 0x5a, 0x2f,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x68, 0x75, 0x61, 0x6e, 0x67,
	0x6a, 0x75, 0x6e, 0x77, 0x65, 0x6e, 0x2f, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x32,
	0x2f, 0x70, 0x62, 0x2f, 0x6d, 0x73, 0x67, 0x3b, 0x6e, 0x70, 0x70, 0x62, 0x6d, 0x73, 0x67, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_msg_proto_rawDescOnce sync.Once
	file_msg_proto_rawDescData = file_msg_proto_rawDesc
)

func file_msg_proto_rawDescGZIP() []byte {
	file_msg_proto_rawDescOnce.Do(func() {
		file_msg_proto_rawDescData = protoimpl.X.CompressGZIP(file_msg_proto_rawDescData)
	})
	return file_msg_proto_rawDescData
}

var file_msg_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_msg_proto_goTypes = []interface{}{
	(*MessageWithMD)(nil),        // 0: nproto.pb.msg.MessageWithMD
	nil,                          // 1: nproto.pb.msg.MessageWithMD.MetaDataEntry
	(*md.MetaDataValueList)(nil), // 2: nproto.pb.md.MetaDataValueList
}
var file_msg_proto_depIdxs = []int32{
	1, // 0: nproto.pb.msg.MessageWithMD.meta_data:type_name -> nproto.pb.msg.MessageWithMD.MetaDataEntry
	2, // 1: nproto.pb.msg.MessageWithMD.MetaDataEntry.value:type_name -> nproto.pb.md.MetaDataValueList
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_msg_proto_init() }
func file_msg_proto_init() {
	if File_msg_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_msg_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MessageWithMD); i {
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
			RawDescriptor: file_msg_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_msg_proto_goTypes,
		DependencyIndexes: file_msg_proto_depIdxs,
		MessageInfos:      file_msg_proto_msgTypes,
	}.Build()
	File_msg_proto = out.File
	file_msg_proto_rawDesc = nil
	file_msg_proto_goTypes = nil
	file_msg_proto_depIdxs = nil
}