package pj

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	. "github.com/huangjunwen/nproto/v2/enc"
)

const (
	// JsonFormat: JavaScript Object Notation format.
	JsonFormat = "json"
	// PbFormat: Protocol Buffer format.
	PbFormat = "pb"
)

var (
	// DefaultJsonEncoder uses json as default encoding format.
	DefaultJsonEncoder Encoder = NewEncoder(&PbJsonEncoder{Format: JsonFormat})

	// DefaultPbEncoder uses pb as default encoding format.
	DefaultPbEncoder Encoder = NewEncoder(&PbJsonEncoder{Format: PbFormat})

	// DefaultPbJsonEncoder uses json/pb as encoding format, format must be specified by target.
	DefaultPbJsonEncoder Encoder = NewEncoder(&PbJsonEncoder{})

	// DefaultPbJsonDecoder uses json/pb as decoding format.
	DefaultPbJsonDecoder Decoder = NewDecoder(&PbJsonDecoder{})
)

// PbJsonEncoder uses json or protobuf to encode data.
type PbJsonEncoder struct {
	// Format specified the default format used for encoding.
	Format string

	// PbMarshalOptions is used for marshal proto.Message using PbFormat.
	PbMarshalOptions proto.MarshalOptions

	// JsonMarshalOptions is used for marshal proto.Message using JsonFormat.
	JsonMarshalOptions protojson.MarshalOptions
}

// PbJsonDecoder uses json or protobuf to decode data.
type PbJsonDecoder struct {
	// PbUnmarshalOptions is used for unmarshal proto.Message using PbFormat.
	PbUnmarshalOptions proto.UnmarshalOptions

	// JsonUnmarshalOptions is used for unmarshal proto.Message using JsonFormat.
	JsonUnmarshalOptions protojson.UnmarshalOptions
}

// EncodeData encodes data, format can be PbFormat or JsonFormat,
// specified by e.Format or target.Format (in order), an error is returned if both are empty.
func (e *PbJsonEncoder) EncodeData(data interface{}, target *RawData) error {

	// Decide target format.
	format := e.Format
	if target.Format != "" {
		format = target.Format
	}

	// PbFormat.
	var (
		b   []byte
		err error
	)

	switch format {
	case PbFormat:
		d, ok := data.(proto.Message)
		if !ok {
			return fmt.Errorf("PbJsonEncoder can't encode %+v using pb format", data)
		}
		b, err = e.PbMarshalOptions.Marshal(d)
		if err != nil {
			return err
		}
		target.Format = PbFormat
		target.Bytes = b
		return nil

	case JsonFormat:
		switch d := data.(type) {
		case proto.Message:
			b, err = e.JsonMarshalOptions.Marshal(d)

		default:
			b, err = json.Marshal(d)
		}
		if err != nil {
			return err
		}
		target.Format = JsonFormat
		target.Bytes = b
		return nil

	default:
		return fmt.Errorf("PbJsonEncoder does not support format %q", format)

	}

}

// DecodeData decodes data:
//   - If src.Format == PbFormat, use protobuf.
//   - If src.Format == JsonFormat, use json.
func (e *PbJsonDecoder) DecodeData(src *RawData, data interface{}) error {

	switch src.Format {
	case PbFormat:
		d, ok := data.(proto.Message)
		if !ok {
			return fmt.Errorf("PbJsonDecoder can't decode %+v using pb format", data)
		}
		return e.PbUnmarshalOptions.Unmarshal(src.Bytes, d)

	case JsonFormat:
		switch d := data.(type) {
		case proto.Message:
			return e.JsonUnmarshalOptions.Unmarshal(src.Bytes, d)

		default:
			return json.Unmarshal(src.Bytes, data)
		}

	default:
		return fmt.Errorf("PbJsonDecoder does not support format %q", src.Format)
	}

}
