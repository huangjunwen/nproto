package enc

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
	MsgData  []byte
	MetaData nproto.MetaData
}
