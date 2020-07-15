// Package pbenc implements an Encoder which uses protobuf to encode/decode data.
package pbenc

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/huangjunwen/nproto/v2/enc"
)

const (
	// Name of the encoder.
	Name = "pb"
)

// PbEncoder uses protobuf to encode/decode data.
type PbEncoder struct {
	PbMarshalOptions   proto.MarshalOptions
	PbUnmarshalOptions proto.UnmarshalOptions
}

var (
	// Default is a PbEncoder with default options.
	Default             = enc.NewEncoder(&PbEncoder{})
	_       enc.Encoder = (*PbEncoder)(nil)
)

// EncoderName returns Name.
func (e *PbEncoder) EncoderName() string {
	return Name
}

// EncodeData accepts proto.Message.
func (e *PbEncoder) EncodeData(data interface{}, w *[]byte) error {

	switch d := data.(type) {
	case proto.Message:
		b, err := e.PbMarshalOptions.Marshal(d)
		if err != nil {
			return err
		}
		*w = b
		return nil

	default:
		return fmt.Errorf("PbEncoder can't encode %#v", data)
	}

}

// DecodeData accepts proto.Message.
func (e *PbEncoder) DecodeData(r []byte, data interface{}) error {

	switch d := data.(type) {
	case proto.Message:
		return e.PbUnmarshalOptions.Unmarshal(r, d)

	default:
		return fmt.Errorf("PbEncoder can't decode to %#v", data)
	}

}
