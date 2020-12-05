// Code generated by protoc-gen-go. DO NOT EDIT.
// source: trace.proto

package traceapi // import "github.com/huangjunwen/nproto/tests/trace/api"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type RecursiveRequest struct {
	Depth                int32    `protobuf:"varint,1,opt,name=depth,proto3" json:"depth,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RecursiveRequest) Reset()         { *m = RecursiveRequest{} }
func (m *RecursiveRequest) String() string { return proto.CompactTextString(m) }
func (*RecursiveRequest) ProtoMessage()    {}
func (*RecursiveRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_trace_d2feec362b85377c, []int{0}
}
func (m *RecursiveRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RecursiveRequest.Unmarshal(m, b)
}
func (m *RecursiveRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RecursiveRequest.Marshal(b, m, deterministic)
}
func (dst *RecursiveRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RecursiveRequest.Merge(dst, src)
}
func (m *RecursiveRequest) XXX_Size() int {
	return xxx_messageInfo_RecursiveRequest.Size(m)
}
func (m *RecursiveRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RecursiveRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RecursiveRequest proto.InternalMessageInfo

func (m *RecursiveRequest) GetDepth() int32 {
	if m != nil {
		return m.Depth
	}
	return 0
}

type RecursiveReply struct {
	Result               int32    `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RecursiveReply) Reset()         { *m = RecursiveReply{} }
func (m *RecursiveReply) String() string { return proto.CompactTextString(m) }
func (*RecursiveReply) ProtoMessage()    {}
func (*RecursiveReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_trace_d2feec362b85377c, []int{1}
}
func (m *RecursiveReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RecursiveReply.Unmarshal(m, b)
}
func (m *RecursiveReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RecursiveReply.Marshal(b, m, deterministic)
}
func (dst *RecursiveReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RecursiveReply.Merge(dst, src)
}
func (m *RecursiveReply) XXX_Size() int {
	return xxx_messageInfo_RecursiveReply.Size(m)
}
func (m *RecursiveReply) XXX_DiscardUnknown() {
	xxx_messageInfo_RecursiveReply.DiscardUnknown(m)
}

var xxx_messageInfo_RecursiveReply proto.InternalMessageInfo

func (m *RecursiveReply) GetResult() int32 {
	if m != nil {
		return m.Result
	}
	return 0
}

// @@npmsg@@
type RecursiveDepthNegative struct {
	Depth                int32    `protobuf:"varint,1,opt,name=depth,proto3" json:"depth,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RecursiveDepthNegative) Reset()         { *m = RecursiveDepthNegative{} }
func (m *RecursiveDepthNegative) String() string { return proto.CompactTextString(m) }
func (*RecursiveDepthNegative) ProtoMessage()    {}
func (*RecursiveDepthNegative) Descriptor() ([]byte, []int) {
	return fileDescriptor_trace_d2feec362b85377c, []int{2}
}
func (m *RecursiveDepthNegative) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RecursiveDepthNegative.Unmarshal(m, b)
}
func (m *RecursiveDepthNegative) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RecursiveDepthNegative.Marshal(b, m, deterministic)
}
func (dst *RecursiveDepthNegative) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RecursiveDepthNegative.Merge(dst, src)
}
func (m *RecursiveDepthNegative) XXX_Size() int {
	return xxx_messageInfo_RecursiveDepthNegative.Size(m)
}
func (m *RecursiveDepthNegative) XXX_DiscardUnknown() {
	xxx_messageInfo_RecursiveDepthNegative.DiscardUnknown(m)
}

var xxx_messageInfo_RecursiveDepthNegative proto.InternalMessageInfo

func (m *RecursiveDepthNegative) GetDepth() int32 {
	if m != nil {
		return m.Depth
	}
	return 0
}

func init() {
	proto.RegisterType((*RecursiveRequest)(nil), "huangjunwen.nproto.tests.traceapi.RecursiveRequest")
	proto.RegisterType((*RecursiveReply)(nil), "huangjunwen.nproto.tests.traceapi.RecursiveReply")
	proto.RegisterType((*RecursiveDepthNegative)(nil), "huangjunwen.nproto.tests.traceapi.RecursiveDepthNegative")
}

func init() { proto.RegisterFile("trace.proto", fileDescriptor_trace_d2feec362b85377c) }

var fileDescriptor_trace_d2feec362b85377c = []byte{
	// 213 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x8e, 0xb1, 0x4b, 0xc5, 0x30,
	0x10, 0xc6, 0x79, 0x43, 0x1f, 0x78, 0x82, 0x48, 0x90, 0x22, 0x4e, 0xda, 0xa9, 0xd3, 0x05, 0x2d,
	0x88, 0xe0, 0x26, 0xce, 0x0e, 0xc5, 0xc9, 0x2d, 0xad, 0x47, 0x1b, 0xa9, 0x69, 0x4c, 0x2e, 0x95,
	0x82, 0x7f, 0xbc, 0x34, 0xad, 0xa5, 0x88, 0x20, 0x6f, 0xfc, 0xee, 0x7e, 0xbf, 0x8f, 0x0f, 0x8e,
	0xd9, 0xa9, 0x9a, 0xd0, 0xba, 0x9e, 0x7b, 0x71, 0xd5, 0x06, 0x65, 0x9a, 0xb7, 0x60, 0x3e, 0xc9,
	0xa0, 0x89, 0x37, 0x64, 0xf2, 0xec, 0x31, 0x52, 0xca, 0xea, 0x2c, 0x87, 0xd3, 0x92, 0xea, 0xe0,
	0xbc, 0x1e, 0xa8, 0xa4, 0x8f, 0x40, 0x9e, 0xc5, 0x19, 0x24, 0xaf, 0x64, 0xb9, 0x3d, 0xdf, 0x5d,
	0xee, 0xf2, 0xa4, 0x9c, 0x43, 0x96, 0xc3, 0xc9, 0x86, 0xb4, 0xdd, 0x28, 0x52, 0xd8, 0x3b, 0xf2,
	0xa1, 0xe3, 0x05, 0x5c, 0x52, 0x86, 0x90, 0xae, 0xe4, 0xe3, 0xe4, 0x3e, 0x51, 0xa3, 0x58, 0x0f,
	0xf4, 0x77, 0xf3, 0xcd, 0x17, 0x24, 0xcf, 0xd3, 0x1e, 0xe1, 0xe1, 0x68, 0x15, 0x45, 0x81, 0xff,
	0xae, 0xc7, 0xdf, 0xd3, 0x2f, 0xae, 0x0f, 0x93, 0x6c, 0x37, 0x3e, 0xdc, 0xbd, 0xdc, 0x36, 0x9a,
	0xdb, 0x50, 0x61, 0xdd, 0xbf, 0xcb, 0x8d, 0x2e, 0x67, 0x5d, 0x46, 0x5d, 0x46, 0x5d, 0x2a, 0xab,
	0xef, 0x7f, 0x8a, 0xaa, 0x7d, 0xfc, 0x17, 0xdf, 0x01, 0x00, 0x00, 0xff, 0xff, 0xcc, 0xcf, 0x05,
	0xbe, 0x74, 0x01, 0x00, 0x00,
}