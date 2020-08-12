// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.7.1
// source: pb.proto

package sendmail

import (
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
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

// Status indicates the sending status.
type EmailEntry_Status int32

const (
	EmailEntry_SENDING EmailEntry_Status = 0
	EmailEntry_SUCCESS EmailEntry_Status = 1
	EmailEntry_FAILED  EmailEntry_Status = 2
)

// Enum value maps for EmailEntry_Status.
var (
	EmailEntry_Status_name = map[int32]string{
		0: "SENDING",
		1: "SUCCESS",
		2: "FAILED",
	}
	EmailEntry_Status_value = map[string]int32{
		"SENDING": 0,
		"SUCCESS": 1,
		"FAILED":  2,
	}
)

func (x EmailEntry_Status) Enum() *EmailEntry_Status {
	p := new(EmailEntry_Status)
	*p = x
	return p
}

func (x EmailEntry_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EmailEntry_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_pb_proto_enumTypes[0].Descriptor()
}

func (EmailEntry_Status) Type() protoreflect.EnumType {
	return &file_pb_proto_enumTypes[0]
}

func (x EmailEntry_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EmailEntry_Status.Descriptor instead.
func (EmailEntry_Status) EnumDescriptor() ([]byte, []int) {
	return file_pb_proto_rawDescGZIP(), []int{1, 0}
}

type EmailEntries struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Entries []*EmailEntry `protobuf:"bytes,1,rep,name=entries,proto3" json:"entries,omitempty"`
}

func (x *EmailEntries) Reset() {
	*x = EmailEntries{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EmailEntries) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmailEntries) ProtoMessage() {}

func (x *EmailEntries) ProtoReflect() protoreflect.Message {
	mi := &file_pb_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EmailEntries.ProtoReflect.Descriptor instead.
func (*EmailEntries) Descriptor() ([]byte, []int) {
	return file_pb_proto_rawDescGZIP(), []int{0}
}

func (x *EmailEntries) GetEntries() []*EmailEntry {
	if x != nil {
		return x.Entries
	}
	return nil
}

// @@npmsg@@
type EmailEntry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Id is unique id of the email entry.
	Id     int64             `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Status EmailEntry_Status `protobuf:"varint,2,opt,name=status,proto3,enum=nproto.demo.sendmail.EmailEntry_Status" json:"status,omitempty"`
	// CreatedAt is the time creating.
	CreatedAt string `protobuf:"bytes,3,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	// EndedAt is the time success/failed.
	EndedAt string `protobuf:"bytes,4,opt,name=ended_at,json=endedAt,proto3" json:"ended_at,omitempty"`
	// FailedReason is sending error when failed.
	FailedReason string `protobuf:"bytes,5,opt,name=failed_reason,json=failedReason,proto3" json:"failed_reason,omitempty"`
	// Email is the concrete content.
	Email *Email `protobuf:"bytes,6,opt,name=email,proto3" json:"email,omitempty"`
}

func (x *EmailEntry) Reset() {
	*x = EmailEntry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EmailEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmailEntry) ProtoMessage() {}

func (x *EmailEntry) ProtoReflect() protoreflect.Message {
	mi := &file_pb_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EmailEntry.ProtoReflect.Descriptor instead.
func (*EmailEntry) Descriptor() ([]byte, []int) {
	return file_pb_proto_rawDescGZIP(), []int{1}
}

func (x *EmailEntry) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *EmailEntry) GetStatus() EmailEntry_Status {
	if x != nil {
		return x.Status
	}
	return EmailEntry_SENDING
}

func (x *EmailEntry) GetCreatedAt() string {
	if x != nil {
		return x.CreatedAt
	}
	return ""
}

func (x *EmailEntry) GetEndedAt() string {
	if x != nil {
		return x.EndedAt
	}
	return ""
}

func (x *EmailEntry) GetFailedReason() string {
	if x != nil {
		return x.FailedReason
	}
	return ""
}

func (x *EmailEntry) GetEmail() *Email {
	if x != nil {
		return x.Email
	}
	return nil
}

type Email struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// To email address.
	ToAddr string `protobuf:"bytes,1,opt,name=to_addr,json=toAddr,proto3" json:"to_addr,omitempty"`
	// To name.
	ToName string `protobuf:"bytes,2,opt,name=to_name,json=toName,proto3" json:"to_name,omitempty"`
	// Subject of email.
	Subject string `protobuf:"bytes,10,opt,name=subject,proto3" json:"subject,omitempty"`
	// ContentType is the mime type of email content. (e.g. text/html)
	ContentType string `protobuf:"bytes,11,opt,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
	// Content is email content.
	Content string `protobuf:"bytes,12,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *Email) Reset() {
	*x = Email{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pb_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Email) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Email) ProtoMessage() {}

func (x *Email) ProtoReflect() protoreflect.Message {
	mi := &file_pb_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Email.ProtoReflect.Descriptor instead.
func (*Email) Descriptor() ([]byte, []int) {
	return file_pb_proto_rawDescGZIP(), []int{2}
}

func (x *Email) GetToAddr() string {
	if x != nil {
		return x.ToAddr
	}
	return ""
}

func (x *Email) GetToName() string {
	if x != nil {
		return x.ToName
	}
	return ""
}

func (x *Email) GetSubject() string {
	if x != nil {
		return x.Subject
	}
	return ""
}

func (x *Email) GetContentType() string {
	if x != nil {
		return x.ContentType
	}
	return ""
}

func (x *Email) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

var File_pb_proto protoreflect.FileDescriptor

var file_pb_proto_rawDesc = []byte{
	0x0a, 0x08, 0x70, 0x62, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14, 0x6e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x64, 0x65, 0x6d, 0x6f, 0x2e, 0x73, 0x65, 0x6e, 0x64, 0x6d, 0x61, 0x69, 0x6c,
	0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x4a, 0x0a,
	0x0c, 0x45, 0x6d, 0x61, 0x69, 0x6c, 0x45, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x12, 0x3a, 0x0a,
	0x07, 0x65, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x20,
	0x2e, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x64, 0x65, 0x6d, 0x6f, 0x2e, 0x73, 0x65, 0x6e,
	0x64, 0x6d, 0x61, 0x69, 0x6c, 0x2e, 0x45, 0x6d, 0x61, 0x69, 0x6c, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x07, 0x65, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x22, 0x9f, 0x02, 0x0a, 0x0a, 0x45, 0x6d,
	0x61, 0x69, 0x6c, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x69, 0x64, 0x12, 0x3f, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x27, 0x2e, 0x6e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x64, 0x65, 0x6d, 0x6f, 0x2e, 0x73, 0x65, 0x6e, 0x64, 0x6d, 0x61, 0x69, 0x6c, 0x2e,
	0x45, 0x6d, 0x61, 0x69, 0x6c, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x63,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x65, 0x6e, 0x64, 0x65,
	0x64, 0x5f, 0x61, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x65, 0x6e, 0x64, 0x65,
	0x64, 0x41, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x66, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x5f, 0x72, 0x65,
	0x61, 0x73, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x66, 0x61, 0x69, 0x6c,
	0x65, 0x64, 0x52, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12, 0x31, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69,
	0x6c, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x64, 0x65, 0x6d, 0x6f, 0x2e, 0x73, 0x65, 0x6e, 0x64, 0x6d, 0x61, 0x69, 0x6c, 0x2e, 0x45,
	0x6d, 0x61, 0x69, 0x6c, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x22, 0x2e, 0x0a, 0x06, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x45, 0x4e, 0x44, 0x49, 0x4e, 0x47,
	0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x55, 0x43, 0x43, 0x45, 0x53, 0x53, 0x10, 0x01, 0x12,
	0x0a, 0x0a, 0x06, 0x46, 0x41, 0x49, 0x4c, 0x45, 0x44, 0x10, 0x02, 0x22, 0x90, 0x01, 0x0a, 0x05,
	0x45, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x17, 0x0a, 0x07, 0x74, 0x6f, 0x5f, 0x61, 0x64, 0x64, 0x72,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x74, 0x6f, 0x41, 0x64, 0x64, 0x72, 0x12, 0x17,
	0x0a, 0x07, 0x74, 0x6f, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x74, 0x6f, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x62, 0x6a, 0x65,
	0x63, 0x74, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x73, 0x75, 0x62, 0x6a, 0x65, 0x63,
	0x74, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x54, 0x79, 0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18,
	0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x32, 0x98,
	0x01, 0x0a, 0x0b, 0x53, 0x65, 0x6e, 0x64, 0x4d, 0x61, 0x69, 0x6c, 0x53, 0x76, 0x63, 0x12, 0x45,
	0x0a, 0x04, 0x53, 0x65, 0x6e, 0x64, 0x12, 0x1b, 0x2e, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e,
	0x64, 0x65, 0x6d, 0x6f, 0x2e, 0x73, 0x65, 0x6e, 0x64, 0x6d, 0x61, 0x69, 0x6c, 0x2e, 0x45, 0x6d,
	0x61, 0x69, 0x6c, 0x1a, 0x20, 0x2e, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x64, 0x65, 0x6d,
	0x6f, 0x2e, 0x73, 0x65, 0x6e, 0x64, 0x6d, 0x61, 0x69, 0x6c, 0x2e, 0x45, 0x6d, 0x61, 0x69, 0x6c,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x42, 0x0a, 0x04, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x16, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x22, 0x2e, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x64,
	0x65, 0x6d, 0x6f, 0x2e, 0x73, 0x65, 0x6e, 0x64, 0x6d, 0x61, 0x69, 0x6c, 0x2e, 0x45, 0x6d, 0x61,
	0x69, 0x6c, 0x45, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x42, 0x39, 0x5a, 0x37, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x68, 0x75, 0x61, 0x6e, 0x67, 0x6a, 0x75, 0x6e,
	0x77, 0x65, 0x6e, 0x2f, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x32, 0x2f, 0x64, 0x65,
	0x6d, 0x6f, 0x2f, 0x73, 0x65, 0x6e, 0x64, 0x6d, 0x61, 0x69, 0x6c, 0x3b, 0x73, 0x65, 0x6e, 0x64,
	0x6d, 0x61, 0x69, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_pb_proto_rawDescOnce sync.Once
	file_pb_proto_rawDescData = file_pb_proto_rawDesc
)

func file_pb_proto_rawDescGZIP() []byte {
	file_pb_proto_rawDescOnce.Do(func() {
		file_pb_proto_rawDescData = protoimpl.X.CompressGZIP(file_pb_proto_rawDescData)
	})
	return file_pb_proto_rawDescData
}

var file_pb_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_pb_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_pb_proto_goTypes = []interface{}{
	(EmailEntry_Status)(0), // 0: nproto.demo.sendmail.EmailEntry.Status
	(*EmailEntries)(nil),   // 1: nproto.demo.sendmail.EmailEntries
	(*EmailEntry)(nil),     // 2: nproto.demo.sendmail.EmailEntry
	(*Email)(nil),          // 3: nproto.demo.sendmail.Email
	(*empty.Empty)(nil),    // 4: google.protobuf.Empty
}
var file_pb_proto_depIdxs = []int32{
	2, // 0: nproto.demo.sendmail.EmailEntries.entries:type_name -> nproto.demo.sendmail.EmailEntry
	0, // 1: nproto.demo.sendmail.EmailEntry.status:type_name -> nproto.demo.sendmail.EmailEntry.Status
	3, // 2: nproto.demo.sendmail.EmailEntry.email:type_name -> nproto.demo.sendmail.Email
	3, // 3: nproto.demo.sendmail.SendMailSvc.Send:input_type -> nproto.demo.sendmail.Email
	4, // 4: nproto.demo.sendmail.SendMailSvc.List:input_type -> google.protobuf.Empty
	2, // 5: nproto.demo.sendmail.SendMailSvc.Send:output_type -> nproto.demo.sendmail.EmailEntry
	1, // 6: nproto.demo.sendmail.SendMailSvc.List:output_type -> nproto.demo.sendmail.EmailEntries
	5, // [5:7] is the sub-list for method output_type
	3, // [3:5] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_pb_proto_init() }
func file_pb_proto_init() {
	if File_pb_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pb_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EmailEntries); i {
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
		file_pb_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EmailEntry); i {
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
		file_pb_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Email); i {
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
			RawDescriptor: file_pb_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_pb_proto_goTypes,
		DependencyIndexes: file_pb_proto_depIdxs,
		EnumInfos:         file_pb_proto_enumTypes,
		MessageInfos:      file_pb_proto_msgTypes,
	}.Build()
	File_pb_proto = out.File
	file_pb_proto_rawDesc = nil
	file_pb_proto_goTypes = nil
	file_pb_proto_depIdxs = nil
}
