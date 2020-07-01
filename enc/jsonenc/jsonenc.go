package jsonenc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	. "github.com/huangjunwen/nproto/v2/enc"
)

// Name of this encoder.
const Name = "nproto.json"

// JsonEncoder uses json to encode/decode payload. Payloads must be
// proto.Message or *RawPayload or JsonIOMarshaler/JsonIOUnmarshaler or
// any json marshalable/unmarshalable object.
type JsonEncoder struct {
	PbMarshalOptions   protojson.MarshalOptions
	PbUnmarshalOptions protojson.UnmarshalOptions
}

var (
	// Default is a JsonEncoder with default options.
	Default         = &JsonEncoder{}
	_       Encoder = (*JsonEncoder)(nil)
)

func (e *JsonEncoder) Name() string {
	return Name
}

func (e *JsonEncoder) EncodePayload(w io.Writer, payload interface{}) error {

	if m, ok := payload.(JsonIOMarshaler); ok {
		return m.MarshalJSONToWriter(w)
	}

	var (
		b   []byte
		err error
	)

	switch m := payload.(type) {
	case *RawPayload:
		if m.EncoderName != Name {
			goto WRONG_PAYLOAD
		}
		b = m.Data

	case RawPayload:
		if m.EncoderName != Name {
			goto WRONG_PAYLOAD
		}
		b = m.Data

	case proto.Message:
		b, err = e.PbMarshalOptions.Marshal(m)

	default:
		b, err = json.Marshal(m)
	}

	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err

WRONG_PAYLOAD:
	return fmt.Errorf("JsonEncoder can't encode %#v", payload)
}

func (e *JsonEncoder) DecodePayload(r io.Reader, payload interface{}) error {

	if m, ok := payload.(JsonIOUnmarshaler); ok {
		return m.UnmarshalJSONFromReader(r)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	switch m := payload.(type) {
	case *RawPayload:
		m.EncoderName = Name
		m.Data = b
		return nil

	case proto.Message:
		return e.PbUnmarshalOptions.Unmarshal(b, m)

	default:
		return json.Unmarshal(b, m)
	}

}
