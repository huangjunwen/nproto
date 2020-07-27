package binlogmsg

import (
	"context"

	"github.com/huangjunwen/golibs/sqlh"
	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc/pjenc"
	. "github.com/huangjunwen/nproto/v2/msg"
)

func PbJsonPublisher(q sqlh.Queryer, schema, table string) MsgPublisherFunc {

	pbPublisher := NewMsgPublisher(pjenc.DefaultPbEncoder, q, schema, table)
	jsonPublisher := NewMsgPublisher(pjenc.DefaultJsonEncoder, q, schema, table)

	return func(ctx context.Context, spec MsgSpec, msg interface{}) error {
		if _, ok := spec.MsgValue().(proto.Message); ok {
			return pbPublisher(ctx, spec, msg)
		}
		return jsonPublisher(ctx, spec, msg)
	}
}
