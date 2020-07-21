package natsrpc

import (
	"reflect"

	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc/pjenc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

var (
	protoMessageType = reflect.TypeOf((*proto.Message)(nil)).Elem()
)

func PbJsonServer(sc *ServerConn) RPCServerFunc {
	return sc.Server(pjenc.DefaultPjDecoder, pjenc.DefaultPjEncoder)
}

func PbJsonClient(cc *ClientConn) RPCClientFunc {

	pbClient := cc.Client(pjenc.DefaultPbEncoder, pjenc.DefaultPjDecoder)
	jsonClient := cc.Client(pjenc.DefaultJsonEncoder, pjenc.DefaultPjDecoder)

	return func(spec RPCSpec) RPCHandler {
		// If both input/output are proto.Message, then use pbClient, otherwise use jsonClient.
		if spec.InputType().Implements(protoMessageType) && spec.OutputType().Implements(protoMessageType) {
			return pbClient(spec)
		}
		return jsonClient(spec)
	}

}
