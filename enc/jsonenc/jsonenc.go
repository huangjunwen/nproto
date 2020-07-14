// Package jsonenc implements an Encoder which uses json to encode/decode data.
package jsonenc

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc"
)

// JsonEncoder uses json to encode/decode data.
type JsonEncoder struct {
	Name               string
	PbMarshalOptions   protojson.MarshalOptions
	PbUnmarshalOptions protojson.UnmarshalOptions
}

var (
	// Default is a JsonEncoder with default options.
	Default = enc.NewEncoder(&JsonEncoder{
		Name: "json",
	})
	_ enc.Encoder = (*JsonEncoder)(nil)
)

// EncoderName returns e.Name.
func (e *JsonEncoder) EncoderName() string {
	return e.Name
}

// EncodeData accepts proto.Message, or any other json marshalable data.
func (e *JsonEncoder) EncodeData(data interface{}, w *[]byte) error {

	var (
		b   []byte
		err error
	)

	switch d := data.(type) {
	case proto.Message:
		b, err = e.PbMarshalOptions.Marshal(d)

	default:
		b, err = json.Marshal(d)
	}

	if err != nil {
		return err
	}
	*w = b
	return nil

}

// DecodeData accepts proto.Message or any other json unmarshalable data.
func (e *JsonEncoder) DecodeData(r []byte, data interface{}) error {

	switch d := data.(type) {
	case proto.Message:
		return e.PbUnmarshalOptions.Unmarshal(r, d)

	default:
		return json.Unmarshal(r, d)
	}

}
