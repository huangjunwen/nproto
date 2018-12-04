package enc

import (
	"time"

	"github.com/huangjunwen/nproto/nproto"

	"github.com/golang/protobuf/proto"
)

// RPCRequest is the request of an rpc.
type RPCRequest struct {
	// Param is the parameter of this rpc.
	// NOTE: Must filled with an empty proto message before decoding.
	// Otherwise the encoder can't determine which type to decode.
	Param proto.Message
	// MetaData is a dict containing extra context information.
	MetaData nproto.MetaData
	// Timeout is timeout of this RPC if > 0.
	Timeout time.Duration
}

// RPCReply is the reply of an rpc.
type RPCReply struct {
	// Result is the normal result of this rpc. Must set to nil if there is an error.
	// NOTE: Must filled with an empty proto message before decoding.
	// Otherwise the encoder can't determine the which type to decode.
	Result proto.Message
	// Error is the error result of this rpc. Must set to nil if there is no error.
	Error error
}

// RPCServerEncoder is the server-side encoder.
type RPCServerEncoder interface {
	// DecodeRequest decodes request from data.
	DecodeRequest(data []byte, req *RPCRequest) error
	// EncodeReply encodes reply to data.
	EncodeReply(reply *RPCReply) ([]byte, error)
}

// RPCClientEncoder is the client-side encoder.
type RPCClientEncoder interface {
	// EncodeRequest encodes request to data.
	EncodeRequest(req *RPCRequest) ([]byte, error)
	// DecodeReply decodes reply from data.
	DecodeReply(data []byte, reply *RPCReply) error
}
