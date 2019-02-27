package enc

import (
	"github.com/golang/protobuf/proto"

	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/pb"
)

// PBMsgPayloadEncoder is MsgPayloadEncoder using protobuf.
type PBMsgPayloadEncoder struct{}

// PBMsgPayloadDecoder is MsgPayloadDecoder using protobuf.
type PBMsgPayloadDecoder struct{}

var (
	_ MsgPayloadEncoder = PBMsgPayloadEncoder{}
	_ MsgPayloadDecoder = PBMsgPayloadDecoder{}
)

// EncodePayload implements MsgPayloadEncoder interface.
func (encoder PBMsgPayloadEncoder) EncodePayload(payload *MsgPayload) ([]byte, error) {
	p := &pb.MsgPayload{
		MsgData: payload.MsgData,
	}

	// Meta data.
	if payload.MD != nil {
		payload.MD.Keys(func(key string) error {
			p.MetaData = append(p.MetaData, &pb.MetaDataKV{
				Key:    key,
				Values: payload.MD.Values(key),
			})
			return nil
		})
	}

	// Encode payload.
	return proto.Marshal(p)
}

// DecodePayload implements MsgSubscriberEncoder interface.
func (decoder PBMsgPayloadDecoder) DecodePayload(data []byte, payload *MsgPayload) error {
	// Decode data.
	p := &pb.MsgPayload{}
	if err := proto.Unmarshal(data, p); err != nil {
		return err
	}

	// Msg data.
	payload.MsgData = p.MsgData

	// Meta data.
	if len(p.MetaData) != 0 {
		md := nproto.MetaData{}
		for _, kv := range p.MetaData {
			md[kv.Key] = kv.Values
		}
		payload.MD = md
	}
	return nil
}
