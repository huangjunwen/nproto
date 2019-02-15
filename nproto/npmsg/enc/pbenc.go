//go:generate protoc --go_out=paths=source_relative:. pbenc.proto

package enc

import (
	"github.com/golang/protobuf/proto"

	"github.com/huangjunwen/nproto/nproto"
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
	p := &PBPayload{
		MsgData: payload.MsgData,
	}
	if len(payload.MetaData) != 0 {
		for key, vals := range payload.MetaData {
			p.MetaData = append(p.MetaData, &MetaDataKV{
				Key:    key,
				Values: vals,
			})
		}
	}
	return proto.Marshal(p)
}

// DecodePayload implements MsgSubscriberEncoder interface.
func (decoder PBMsgPayloadDecoder) DecodePayload(data []byte, payload *MsgPayload) error {
	p := &PBPayload{}
	if err := proto.Unmarshal(data, p); err != nil {
		return err
	}
	payload.MsgData = p.MsgData
	if len(p.MetaData) != 0 {
		payload.MetaData = nproto.MetaData{}
		for _, kv := range p.MetaData {
			payload.MetaData[kv.Key] = kv.Values
		}
	}
	return nil
}
