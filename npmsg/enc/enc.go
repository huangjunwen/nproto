package enc

import (
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
	// Passthru is an optional context dict passing around.
	Passthru map[string]string
}
