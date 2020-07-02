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

// JsonEncoder uses json to encode/decode data. Accept
// proto.Message or *RawData or JsonIOMarshaler/JsonIOUnmarshaler or
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

func (e *JsonEncoder) EncodeData(w io.Writer, data interface{}) error {

	if m, ok := data.(JsonIOMarshaler); ok {
		return m.MarshalJSONToWriter(w)
	}

	var (
		b   []byte
		err error
	)

	switch m := data.(type) {
	case *RawData:
		if m.EncoderName != Name {
			goto WRONG_DATA
		}
		b = m.Bytes

	case RawData:
		if m.EncoderName != Name {
			goto WRONG_DATA
		}
		b = m.Bytes

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

WRONG_DATA:
	return fmt.Errorf("JsonEncoder can't encode %#v", data)
}

func (e *JsonEncoder) DecodeData(r io.Reader, data interface{}) error {

	if m, ok := data.(JsonIOUnmarshaler); ok {
		return m.UnmarshalJSONFromReader(r)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	switch m := data.(type) {
	case *RawData:
		m.EncoderName = Name
		m.Bytes = b
		return nil

	case proto.Message:
		return e.PbUnmarshalOptions.Unmarshal(b, m)

	default:
		return json.Unmarshal(b, m)
	}

}
