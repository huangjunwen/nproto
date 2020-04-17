package msgenc

import (
	"github.com/huangjunwen/nproto/nproto"
)

// MsgPayloadEncoder is used to encode MsgPayload.
type MsgPayloadEncoder interface {
	// EncodePayload encodes payload to data.
	EncodePayload(payload *MsgPayload) ([]byte, error)
}

// MsgPayloadDecoder is used to decode MsgPayload.
type MsgPayloadDecoder interface {
	// DecodePayloa decodes payload from data.
	DecodePayload(data []byte, payload *MsgPayload) error
}

// MsgPayload is the payload.
type MsgPayload struct {
	// MsgData is serialized msg.
	MsgData []byte
	// MD is a dict containing extra context information. Maybe nil.
	MD nproto.MD
}
