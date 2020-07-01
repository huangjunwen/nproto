package pbenc

import (
	"fmt"
	"io"
	"io/ioutil"

	"google.golang.org/protobuf/proto"

	. "github.com/huangjunwen/nproto/v2/enc"
)

// Name of this encoder.
const Name = "nproto.pb"

// PbEncoder uses protobuf to encode/decode payload. Payloads must be
// proto.Message or *RawPayload.
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

func (e *PbEncoder) EncodePayload(w io.Writer, payload interface{}) error {

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
		goto WRONG_PAYLOAD
	}

	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err

WRONG_PAYLOAD:
	return fmt.Errorf("PbEncoder can't encode %#v", payload)

}

func (e *PbEncoder) DecodePayload(r io.Reader, payload interface{}) error {

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
		return fmt.Errorf("PbEncoder can't decode to %#v", payload)
	}

}
