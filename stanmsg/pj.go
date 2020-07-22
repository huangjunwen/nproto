package stanmsg

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc/pjenc"
	. "github.com/huangjunwen/nproto/v2/msg"
)

func PbJsonPublisher(dc *DurConn) MsgAsyncPublisherFunc {

	pbPublisher := dc.Publisher(pjenc.DefaultPbEncoder)
	jsonPublisher := dc.Publisher(pjenc.DefaultJsonEncoder)

	return func(ctx context.Context, spec MsgSpec, msg interface{}, cb func(error)) error {
		// If msg is proto.Message, then use pbPublisher, otherwise jsonPublisher.
		if _, ok := spec.MsgValue().(proto.Message); ok {
			return pbPublisher(ctx, spec, msg, cb)
		}
		return jsonPublisher(ctx, spec, msg, cb)
	}

}

func PbJsonSubscriber(dc *DurConn) MsgSubscriberFunc {
	return dc.Subscriber(pjenc.DefaultPjDecoder)
}
