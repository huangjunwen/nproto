package enc

import (
	"github.com/huangjunwen/nproto/nproto"

	"github.com/golang/protobuf/proto"
)

// MsgPublisherEncoder is the publisher-side encoder.
type MsgPublisherEncoder interface {
	// EncodePayload encodes payload to data.
	EncodePayload(payload *MsgPayload) ([]byte, error)
}

// MsgSubscriberEncoder is the subscriber-side encoder.
type MsgSubscriberEncoder interface {
	// DecodePayloa decodes payload from data.
	DecodePayload(data []byte, payload *MsgPayload) error
}

// MsgPayload is the payload.
type MsgPayload struct {
	// Msg is the published message.
	Msg proto.Message
	// MetaData is an optional dict.
	MetaData nproto.MetaData
}
