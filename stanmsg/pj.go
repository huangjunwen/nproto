package stanmsg

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc/pjenc"
	. "github.com/huangjunwen/nproto/v2/msg"
)

// PbJsonPublisher creates a msg publisher using protobuf or json for encoding:
//   - If msg is proto.Message, then use protobuf.
//   - Otherwise use json.
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

// PbJsonSubscriber creates a msg subscriber using protobuf or json for decoding.
func PbJsonSubscriber(dc *DurConn) MsgSubscriberFunc {
	return dc.Subscriber(pjenc.DefaultPjDecoder)
}
