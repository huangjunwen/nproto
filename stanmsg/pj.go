package stanmsg

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc/pjenc"
	. "github.com/huangjunwen/nproto/v2/msg"
)

// NewPbJsonPublisher creates a msg publisher using protobuf or json for encoding:
//   - If msg is proto.Message, then use protobuf.
//   - Otherwise use json.
func NewPbJsonPublisher(dc *DurConn) MsgAsyncPublisherFunc {

	pbPublisher := dc.NewPublisher(pjenc.DefaultPbEncoder)
	jsonPublisher := dc.NewPublisher(pjenc.DefaultJsonEncoder)

	return func(ctx context.Context, spec MsgSpec, msg interface{}, cb func(error)) error {
		// If msg is proto.Message, then use pbPublisher, otherwise jsonPublisher.
		if _, ok := spec.MsgValue().(proto.Message); ok {
			return pbPublisher(ctx, spec, msg, cb)
		}
		return jsonPublisher(ctx, spec, msg, cb)
	}

}

// NewPbJsonSubscriber creates a msg subscriber using protobuf or json for decoding.
func NewPbJsonSubscriber(dc *DurConn) MsgSubscriberFunc {
	return dc.NewSubscriber(pjenc.DefaultPjDecoder)
}
