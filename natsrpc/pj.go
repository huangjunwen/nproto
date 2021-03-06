package natsrpc

import (
	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc/pjenc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

// NewPbJsonClient creates an rpc client using protobuf or json for encoding/decoding:
//   - If both input and output are proto.Message, then use protobuf.
//   - Otherwise use json.
func NewPbJsonClient(cc *ClientConn) RPCClientFunc {

	pbClient := cc.NewClient(pjenc.DefaultPbEncoder, pjenc.DefaultPjDecoder)
	jsonClient := cc.NewClient(pjenc.DefaultJsonEncoder, pjenc.DefaultPjDecoder)

	return func(spec RPCSpec) RPCHandler {
		_, ok1 := spec.InputValue().(proto.Message)
		_, ok2 := spec.OutputValue().(proto.Message)
		// If both input/output are proto.Message, then use pbClient, otherwise use jsonClient.
		if ok1 && ok2 {
			return pbClient(spec)
		}
		return jsonClient(spec)
	}

}

// NewPbJsonServer creates an rpc server using protobuf or json for decoding/encoding.
func NewPbJsonServer(sc *ServerConn) RPCServerFunc {
	return sc.NewServer(pjenc.DefaultPjDecoder, pjenc.DefaultPjEncoder)
}
