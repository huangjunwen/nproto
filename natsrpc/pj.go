package natsrpc

import (
	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc/pjenc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

func PbJsonClient(cc *ClientConn) RPCClientFunc {

	pbClient := cc.Client(pjenc.DefaultPbEncoder, pjenc.DefaultPjDecoder)
	jsonClient := cc.Client(pjenc.DefaultJsonEncoder, pjenc.DefaultPjDecoder)

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

func PbJsonServer(sc *ServerConn) RPCServerFunc {
	return sc.Server(pjenc.DefaultPjDecoder, pjenc.DefaultPjEncoder)
}
