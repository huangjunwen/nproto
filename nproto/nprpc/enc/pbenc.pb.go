// Code generated by protoc-gen-go. DO NOT EDIT.
// source: pbenc.proto

package enc

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import duration "github.com/golang/protobuf/ptypes/duration"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// PBRequest is request of a RPC call encoded by protobuf.
type PBRequest struct {
	// Param is protobuf encoded param.
	Param    []byte                  `protobuf:"bytes,1,opt,name=param,proto3" json:"param,omitempty"`
	MetaData []*PBRequest_MetaDataKV `protobuf:"bytes,2,rep,name=meta_data,json=metaData,proto3" json:"meta_data,omitempty"`
	// Timeout sets an optinal timeout for this RPC.
	Timeout              *duration.Duration `protobuf:"bytes,3,opt,name=timeout,proto3" json:"timeout,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *PBRequest) Reset()         { *m = PBRequest{} }
func (m *PBRequest) String() string { return proto.CompactTextString(m) }
func (*PBRequest) ProtoMessage()    {}
func (*PBRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_pbenc_ccc2a9c62adc44f7, []int{0}
}
func (m *PBRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PBRequest.Unmarshal(m, b)
}
func (m *PBRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PBRequest.Marshal(b, m, deterministic)
}
func (dst *PBRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PBRequest.Merge(dst, src)
}
func (m *PBRequest) XXX_Size() int {
	return xxx_messageInfo_PBRequest.Size(m)
}
func (m *PBRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_PBRequest.DiscardUnknown(m)
}

var xxx_messageInfo_PBRequest proto.InternalMessageInfo

func (m *PBRequest) GetParam() []byte {
	if m != nil {
		return m.Param
	}
	return nil
}

func (m *PBRequest) GetMetaData() []*PBRequest_MetaDataKV {
	if m != nil {
		return m.MetaData
	}
	return nil
}

func (m *PBRequest) GetTimeout() *duration.Duration {
	if m != nil {
		return m.Timeout
	}
	return nil
}

// MetaData is optional meta data.
type PBRequest_MetaDataKV struct {
	Key                  string   `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Values               []string `protobuf:"bytes,2,rep,name=values,proto3" json:"values,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PBRequest_MetaDataKV) Reset()         { *m = PBRequest_MetaDataKV{} }
func (m *PBRequest_MetaDataKV) String() string { return proto.CompactTextString(m) }
func (*PBRequest_MetaDataKV) ProtoMessage()    {}
func (*PBRequest_MetaDataKV) Descriptor() ([]byte, []int) {
	return fileDescriptor_pbenc_ccc2a9c62adc44f7, []int{0, 0}
}
func (m *PBRequest_MetaDataKV) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PBRequest_MetaDataKV.Unmarshal(m, b)
}
func (m *PBRequest_MetaDataKV) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PBRequest_MetaDataKV.Marshal(b, m, deterministic)
}
func (dst *PBRequest_MetaDataKV) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PBRequest_MetaDataKV.Merge(dst, src)
}
func (m *PBRequest_MetaDataKV) XXX_Size() int {
	return xxx_messageInfo_PBRequest_MetaDataKV.Size(m)
}
func (m *PBRequest_MetaDataKV) XXX_DiscardUnknown() {
	xxx_messageInfo_PBRequest_MetaDataKV.DiscardUnknown(m)
}

var xxx_messageInfo_PBRequest_MetaDataKV proto.InternalMessageInfo

func (m *PBRequest_MetaDataKV) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *PBRequest_MetaDataKV) GetValues() []string {
	if m != nil {
		return m.Values
	}
	return nil
}

// PBReply is reply of a RPC call encoded by protobuf.
type PBReply struct {
	// Types that are valid to be assigned to Reply:
	//	*PBReply_Result
	//	*PBReply_Error
	Reply                isPBReply_Reply `protobuf_oneof:"reply"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *PBReply) Reset()         { *m = PBReply{} }
func (m *PBReply) String() string { return proto.CompactTextString(m) }
func (*PBReply) ProtoMessage()    {}
func (*PBReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_pbenc_ccc2a9c62adc44f7, []int{1}
}
func (m *PBReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PBReply.Unmarshal(m, b)
}
func (m *PBReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PBReply.Marshal(b, m, deterministic)
}
func (dst *PBReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PBReply.Merge(dst, src)
}
func (m *PBReply) XXX_Size() int {
	return xxx_messageInfo_PBReply.Size(m)
}
func (m *PBReply) XXX_DiscardUnknown() {
	xxx_messageInfo_PBReply.DiscardUnknown(m)
}

var xxx_messageInfo_PBReply proto.InternalMessageInfo

type isPBReply_Reply interface {
	isPBReply_Reply()
}

type PBReply_Result struct {
	Result []byte `protobuf:"bytes,1,opt,name=result,proto3,oneof"`
}
type PBReply_Error struct {
	Error string `protobuf:"bytes,2,opt,name=error,proto3,oneof"`
}

func (*PBReply_Result) isPBReply_Reply() {}
func (*PBReply_Error) isPBReply_Reply()  {}

func (m *PBReply) GetReply() isPBReply_Reply {
	if m != nil {
		return m.Reply
	}
	return nil
}

func (m *PBReply) GetResult() []byte {
	if x, ok := m.GetReply().(*PBReply_Result); ok {
		return x.Result
	}
	return nil
}

func (m *PBReply) GetError() string {
	if x, ok := m.GetReply().(*PBReply_Error); ok {
		return x.Error
	}
	return ""
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*PBReply) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _PBReply_OneofMarshaler, _PBReply_OneofUnmarshaler, _PBReply_OneofSizer, []interface{}{
		(*PBReply_Result)(nil),
		(*PBReply_Error)(nil),
	}
}

func _PBReply_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*PBReply)
	// reply
	switch x := m.Reply.(type) {
	case *PBReply_Result:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		b.EncodeRawBytes(x.Result)
	case *PBReply_Error:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.Error)
	case nil:
	default:
		return fmt.Errorf("PBReply.Reply has unexpected type %T", x)
	}
	return nil
}

func _PBReply_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*PBReply)
	switch tag {
	case 1: // reply.result
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeRawBytes(true)
		m.Reply = &PBReply_Result{x}
		return true, err
	case 2: // reply.error
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Reply = &PBReply_Error{x}
		return true, err
	default:
		return false, nil
	}
}

func _PBReply_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*PBReply)
	// reply
	switch x := m.Reply.(type) {
	case *PBReply_Result:
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(len(x.Result)))
		n += len(x.Result)
	case *PBReply_Error:
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(len(x.Error)))
		n += len(x.Error)
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

func init() {
	proto.RegisterType((*PBRequest)(nil), "enc.PBRequest")
	proto.RegisterType((*PBRequest_MetaDataKV)(nil), "enc.PBRequest.MetaDataKV")
	proto.RegisterType((*PBReply)(nil), "enc.PBReply")
}

func init() { proto.RegisterFile("pbenc.proto", fileDescriptor_pbenc_ccc2a9c62adc44f7) }

var fileDescriptor_pbenc_ccc2a9c62adc44f7 = []byte{
	// 251 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x44, 0x8f, 0xc1, 0x4a, 0xc3, 0x40,
	0x10, 0x86, 0x9b, 0x86, 0x24, 0x66, 0xea, 0x41, 0x16, 0x29, 0xb1, 0x07, 0x09, 0x3d, 0xf5, 0xb4,
	0x05, 0x0b, 0x7d, 0x80, 0x92, 0x43, 0x41, 0x04, 0xd9, 0x83, 0x57, 0x99, 0xb4, 0x63, 0x29, 0x26,
	0xd9, 0xb8, 0x99, 0x15, 0xfa, 0x82, 0x3e, 0x97, 0xec, 0x6e, 0xaa, 0xb7, 0xf9, 0x96, 0xf9, 0xff,
	0xfd, 0x06, 0x66, 0x7d, 0x4d, 0xdd, 0x41, 0xf6, 0x46, 0xb3, 0x16, 0x31, 0x75, 0x87, 0xc5, 0xe3,
	0x49, 0xeb, 0x53, 0x43, 0x6b, 0xff, 0x54, 0xdb, 0x8f, 0xf5, 0xd1, 0x1a, 0xe4, 0xb3, 0xee, 0xc2,
	0xd2, 0xf2, 0x27, 0x82, 0xfc, 0x75, 0xa7, 0xe8, 0xcb, 0xd2, 0xc0, 0xe2, 0x1e, 0x92, 0x1e, 0x0d,
	0xb6, 0x45, 0x54, 0x46, 0xab, 0x5b, 0x15, 0x40, 0x6c, 0x21, 0x6f, 0x89, 0xf1, 0xfd, 0x88, 0x8c,
	0xc5, 0xb4, 0x8c, 0x57, 0xb3, 0xa7, 0x07, 0xe9, 0xfe, 0xf9, 0x0b, 0xca, 0x17, 0x62, 0xac, 0x90,
	0xf1, 0xf9, 0x4d, 0xdd, 0xb4, 0xe3, 0x2c, 0x36, 0x90, 0xf1, 0xb9, 0x25, 0x6d, 0xb9, 0x88, 0xcb,
	0xc8, 0xa7, 0x82, 0x8d, 0xbc, 0xda, 0xc8, 0x6a, 0xb4, 0x51, 0xd7, 0xcd, 0xc5, 0x16, 0xe0, 0xbf,
	0x4c, 0xdc, 0x41, 0xfc, 0x49, 0x17, 0xaf, 0x93, 0x2b, 0x37, 0x8a, 0x39, 0xa4, 0xdf, 0xd8, 0x58,
	0x1a, 0xbc, 0x49, 0xae, 0x46, 0x5a, 0x56, 0x90, 0x39, 0x9d, 0xbe, 0xb9, 0x88, 0x02, 0x52, 0x43,
	0x83, 0x6d, 0x38, 0x9c, 0xb1, 0x9f, 0xa8, 0x91, 0xc5, 0x1c, 0x12, 0x32, 0x46, 0x9b, 0x62, 0xea,
	0x0a, 0xf7, 0x13, 0x15, 0x70, 0x97, 0x41, 0x62, 0x5c, 0xb4, 0x4e, 0xbd, 0xd9, 0xe6, 0x37, 0x00,
	0x00, 0xff, 0xff, 0x3f, 0x3b, 0x7f, 0x70, 0x49, 0x01, 0x00, 0x00,
}
