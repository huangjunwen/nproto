package pbenc

import (
	"fmt"
	"io"
	"io/ioutil"

	"google.golang.org/protobuf/proto"

	. "github.com/huangjunwen/nproto/v2/enc"
)

// Name of this encoder.
const Name = "nproto-pb"

// PbEncoder uses protobuf to encode/decode data. Accept proto.Message or *RawData.
type PbEncoder struct {
	PbMarshalOptions   proto.MarshalOptions
	PbUnmarshalOptions proto.UnmarshalOptions
}

var (
	// Default is a PbEncoder with default options.
	Default         = &PbEncoder{}
	_       Encoder = (*PbEncoder)(nil)
)

func (e *PbEncoder) Name() string {
	return Name
}

func (e *PbEncoder) EncodeData(w io.Writer, data interface{}) error {

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
		goto WRONG_DATA
	}

	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err

WRONG_DATA:
	return fmt.Errorf("PbEncoder can't encode %#v", data)

}

func (e *PbEncoder) DecodeData(r io.Reader, data interface{}) error {

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
		return fmt.Errorf("PbEncoder can't decode to %#v", data)
	}

}
