//go:generate protoc --go_out=. pbenc.proto

package enc

import (
	"github.com/golang/protobuf/proto"
)

// PBPublisherEncoder is MsgPublisherEncoder using protobuf encoding.
type PBPublisherEncoder struct{}

// PBSubscriberEncoder is MsgSubscriberEncoder using protobuf encoding.
type PBSubscriberEncoder struct{}

var (
	_ MsgPublisherEncoder  = PBPublisherEncoder{}
	_ MsgSubscriberEncoder = PBSubscriberEncoder{}
)

// EncodePayload implements MsgPublisherEncoder interface.
func (e PBPublisherEncoder) EncodePayload(payload *MsgPayload) ([]byte, error) {
	var err error
	p := &PBPayload{}

	// Encode Msg.
	p.Msg, err = proto.Marshal(payload.Msg)
	if err != nil {
		return nil, err
	}

	// Optional passthru.
	p.Passthru = payload.Passthru
	return proto.Marshal(p)
}

// DecodePayload implements MsgSubscriberEncoder interface.
func (e PBSubscriberEncoder) DecodePayload(data []byte, payload *MsgPayload) error {
	// Decode payload.
	p := &PBPayload{}
	if err := proto.Unmarshal(data, p); err != nil {
		return err
	}

	// Decode msg.
	if err := proto.Unmarshal(p.Msg, payload.Msg); err != nil {
		return err
	}

	// Optional passthru.
	payload.Passthru = p.Passthru
	return nil
}
