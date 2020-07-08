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

// JsonEncoder uses json to encode/decode data.
type JsonEncoder struct {
	Name               string
	PbMarshalOptions   protojson.MarshalOptions
	PbUnmarshalOptions protojson.UnmarshalOptions
}

var (
	// Default is a JsonEncoder with default options.
	Default = &JsonEncoder{
		Name: "nproto-json",
	}
	_ Encoder = (*JsonEncoder)(nil)
)

// EncoderName returns e.Name.
func (e *JsonEncoder) EncoderName() string {
	return e.Name
}

// EncodeData accepts JsonIOMarshaler/proto.Message,
// *RawData/RawData with same encoder name as the encoder,
// or any other json.Marshalable data.
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
		if m.EncoderName != e.Name {
			goto WRONG_DATA
		}
		b = m.Bytes

	case RawData:
		if m.EncoderName != e.Name {
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

// DecodeData accepts JsonIOUnmarshaler/proto.Message/*RawData,
// or any other json.Unmarshalable data.
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
		m.EncoderName = e.Name
		m.Bytes = b
		return nil

	case proto.Message:
		return e.PbUnmarshalOptions.Unmarshal(b, m)

	default:
		return json.Unmarshal(b, m)
	}

}
