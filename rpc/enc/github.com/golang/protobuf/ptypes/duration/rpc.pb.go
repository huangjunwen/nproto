// Code generated by protoc-gen-go. DO NOT EDIT.
// source: rpc.proto

package duration // import "github.com/golang/protobuf/ptypes/duration"

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

// RPCRequest is request of a RPC call. Similar to jsonrpc.
type RPCRequest struct {
	// Id is the identifier of this RPC call.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Method name of the RPC.
	Method string `protobuf:"bytes,2,opt,name=method,proto3" json:"method,omitempty"`
	// Params of the RPC.
	Params []byte `protobuf:"bytes,3,opt,name=params,proto3" json:"params,omitempty"`
	// Timeout sets and optinal timeout for this RPC.
	Timeout *Duration `protobuf:"bytes,9,opt,name=timeout,proto3" json:"timeout,omitempty"`
	// Context is an optional dict carrying context values.
	Context              map[string]string `protobuf:"bytes,10,rep,name=context,proto3" json:"context,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *RPCRequest) Reset()         { *m = RPCRequest{} }
func (m *RPCRequest) String() string { return proto.CompactTextString(m) }
func (*RPCRequest) ProtoMessage()    {}
func (*RPCRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_rpc_19dcb97d81a37e75, []int{0}
}
func (m *RPCRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RPCRequest.Unmarshal(m, b)
}
func (m *RPCRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RPCRequest.Marshal(b, m, deterministic)
}
func (dst *RPCRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RPCRequest.Merge(dst, src)
}
func (m *RPCRequest) XXX_Size() int {
	return xxx_messageInfo_RPCRequest.Size(m)
}
func (m *RPCRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RPCRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RPCRequest proto.InternalMessageInfo

func (m *RPCRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *RPCRequest) GetMethod() string {
	if m != nil {
		return m.Method
	}
	return ""
}

func (m *RPCRequest) GetParams() []byte {
	if m != nil {
		return m.Params
	}
	return nil
}

func (m *RPCRequest) GetTimeout() *Duration {
	if m != nil {
		return m.Timeout
	}
	return nil
}

func (m *RPCRequest) GetContext() map[string]string {
	if m != nil {
		return m.Context
	}
	return nil
}

// RPCReply is reply of a RPC call. Similar to jsonrpc.
type RPCReply struct {
	// Id is the identifier of this RPC call.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Result is the normal reply of the RPC. Must not set when there is an error.
	Result []byte `protobuf:"bytes,2,opt,name=result,proto3" json:"result,omitempty"`
	// Error is the error reply of the RPC. Must not set when there is no error.
	Error                *RPCReply_RPCErrorReply `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *RPCReply) Reset()         { *m = RPCReply{} }
func (m *RPCReply) String() string { return proto.CompactTextString(m) }
func (*RPCReply) ProtoMessage()    {}
func (*RPCReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_rpc_19dcb97d81a37e75, []int{1}
}
func (m *RPCReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RPCReply.Unmarshal(m, b)
}
func (m *RPCReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RPCReply.Marshal(b, m, deterministic)
}
func (dst *RPCReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RPCReply.Merge(dst, src)
}
func (m *RPCReply) XXX_Size() int {
	return xxx_messageInfo_RPCReply.Size(m)
}
func (m *RPCReply) XXX_DiscardUnknown() {
	xxx_messageInfo_RPCReply.DiscardUnknown(m)
}

var xxx_messageInfo_RPCReply proto.InternalMessageInfo

func (m *RPCReply) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *RPCReply) GetResult() []byte {
	if m != nil {
		return m.Result
	}
	return nil
}

func (m *RPCReply) GetError() *RPCReply_RPCErrorReply {
	if m != nil {
		return m.Error
	}
	return nil
}

type RPCReply_RPCErrorReply struct {
	// Code is the error code.
	Code int32 `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	// Message is the error description.
	Message              string   `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RPCReply_RPCErrorReply) Reset()         { *m = RPCReply_RPCErrorReply{} }
func (m *RPCReply_RPCErrorReply) String() string { return proto.CompactTextString(m) }
func (*RPCReply_RPCErrorReply) ProtoMessage()    {}
func (*RPCReply_RPCErrorReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_rpc_19dcb97d81a37e75, []int{1, 0}
}
func (m *RPCReply_RPCErrorReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RPCReply_RPCErrorReply.Unmarshal(m, b)
}
func (m *RPCReply_RPCErrorReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RPCReply_RPCErrorReply.Marshal(b, m, deterministic)
}
func (dst *RPCReply_RPCErrorReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RPCReply_RPCErrorReply.Merge(dst, src)
}
func (m *RPCReply_RPCErrorReply) XXX_Size() int {
	return xxx_messageInfo_RPCReply_RPCErrorReply.Size(m)
}
func (m *RPCReply_RPCErrorReply) XXX_DiscardUnknown() {
	xxx_messageInfo_RPCReply_RPCErrorReply.DiscardUnknown(m)
}

var xxx_messageInfo_RPCReply_RPCErrorReply proto.InternalMessageInfo

func (m *RPCReply_RPCErrorReply) GetCode() int32 {
	if m != nil {
		return m.Code
	}
	return 0
}

func (m *RPCReply_RPCErrorReply) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func init() {
	proto.RegisterType((*RPCRequest)(nil), "pb.RPCRequest")
	proto.RegisterMapType((map[string]string)(nil), "pb.RPCRequest.ContextEntry")
	proto.RegisterType((*RPCReply)(nil), "pb.RPCReply")
	proto.RegisterType((*RPCReply_RPCErrorReply)(nil), "pb.RPCReply.RPCErrorReply")
}

func init() { proto.RegisterFile("rpc.proto", fileDescriptor_rpc_19dcb97d81a37e75) }

var fileDescriptor_rpc_19dcb97d81a37e75 = []byte{
	// 330 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x91, 0x5d, 0x4b, 0xf3, 0x30,
	0x14, 0xc7, 0x49, 0xf7, 0x6c, 0x7b, 0x76, 0x36, 0x45, 0x82, 0x48, 0xad, 0x20, 0x65, 0x57, 0x45,
	0x24, 0x93, 0x0d, 0x41, 0x06, 0xde, 0x38, 0x77, 0x2f, 0xb9, 0xf4, 0xae, 0x2f, 0xc7, 0xae, 0xd8,
	0x36, 0x31, 0x4d, 0xc4, 0x7e, 0x1e, 0xbf, 0xa2, 0x1f, 0x40, 0x9a, 0xb6, 0x9b, 0x82, 0x77, 0xe7,
	0xff, 0x12, 0xf2, 0x3b, 0x1c, 0x98, 0x28, 0x19, 0x33, 0xa9, 0x84, 0x16, 0xd4, 0x91, 0x91, 0x77,
	0x99, 0x0a, 0x91, 0xe6, 0xb8, 0xb0, 0x4e, 0x64, 0x5e, 0x16, 0x89, 0x51, 0xa1, 0xce, 0x44, 0xd9,
	0x76, 0xe6, 0x5f, 0x04, 0x80, 0x3f, 0x6d, 0x38, 0xbe, 0x19, 0xac, 0x34, 0x3d, 0x06, 0x27, 0x4b,
	0x5c, 0xe2, 0x93, 0x60, 0xc2, 0x9d, 0x2c, 0xa1, 0x67, 0x30, 0x2a, 0x50, 0xef, 0x44, 0xe2, 0x3a,
	0xd6, 0xeb, 0x54, 0xe3, 0xcb, 0x50, 0x85, 0x45, 0xe5, 0x0e, 0x7c, 0x12, 0xcc, 0x78, 0xa7, 0xe8,
	0x0a, 0xc6, 0x3a, 0x2b, 0x50, 0x18, 0xed, 0x4e, 0x7c, 0x12, 0x4c, 0x97, 0xe7, 0xac, 0x05, 0x60,
	0x3d, 0x00, 0x7b, 0xec, 0x00, 0x78, 0xdf, 0xa4, 0xb7, 0x30, 0x8e, 0x45, 0xa9, 0xf1, 0x43, 0xbb,
	0xe0, 0x0f, 0x82, 0xe9, 0xf2, 0x82, 0xc9, 0x88, 0x1d, 0xa8, 0xd8, 0xa6, 0x4d, 0xb7, 0xa5, 0x56,
	0x35, 0xef, 0xbb, 0xde, 0x1a, 0x66, 0x3f, 0x03, 0x7a, 0x02, 0x83, 0x57, 0xac, 0x3b, 0xf8, 0x66,
	0xa4, 0xa7, 0x30, 0x7c, 0x0f, 0x73, 0x83, 0x1d, 0x7c, 0x2b, 0xd6, 0xce, 0x1d, 0x99, 0x7f, 0x12,
	0xf8, 0x6f, 0x3f, 0x90, 0x79, 0xfd, 0xd7, 0xd2, 0x0a, 0x2b, 0x93, 0x6b, 0xfb, 0x6e, 0xc6, 0x3b,
	0x45, 0x6f, 0x60, 0x88, 0x4a, 0x09, 0x65, 0x77, 0x9e, 0x2e, 0xbd, 0x3d, 0xa5, 0xcc, 0xeb, 0x66,
	0xd8, 0x36, 0xa1, 0x55, 0xbc, 0x2d, 0x7a, 0xf7, 0x70, 0xf4, 0xcb, 0xa7, 0x14, 0xfe, 0xc5, 0x22,
	0x41, 0xfb, 0xd9, 0x90, 0xdb, 0x99, 0xba, 0x30, 0x2e, 0xb0, 0xaa, 0xc2, 0xb4, 0xe7, 0xec, 0xe5,
	0xc3, 0xf5, 0xf3, 0x55, 0x9a, 0xe9, 0x9d, 0x89, 0x58, 0x2c, 0x8a, 0x45, 0x2a, 0xf2, 0xb0, 0x4c,
	0x0f, 0x97, 0x94, 0xba, 0x96, 0x58, 0xed, 0x0f, 0x1a, 0x8d, 0x6c, 0xb2, 0xfa, 0x0e, 0x00, 0x00,
	0xff, 0xff, 0x2d, 0x9c, 0x21, 0x71, 0x02, 0x02, 0x00, 0x00,
}