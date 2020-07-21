// Package pjenc contains Encoder/Decoder using pb or json format.
package pjenc

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	npenc "github.com/huangjunwen/nproto/v2/enc"
)

var (
	// DefaultJsonEncoder uses json as default encoding format.
	DefaultJsonEncoder npenc.Encoder = &PbJsonEncoder{Format: npenc.JsonFormat}

	// DefaultPbEncoder uses pb as default encoding format.
	DefaultPbEncoder npenc.Encoder = &PbJsonEncoder{Format: npenc.PbFormat}

	// DefaultPjEncoder uses pb or json as encoding format, format must be specified by target.
	DefaultPjEncoder npenc.Encoder = &PbJsonEncoder{}

	// DefaultPjDecoder uses pb or json as decoding format.
	DefaultPjDecoder npenc.Decoder = &PbJsonDecoder{}
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
// specified by e.Format or targetFormat (in order), an error is returned if both are empty.
func (e *PbJsonEncoder) EncodeData(data interface{}, targetFormat *string, targetBytes *[]byte) error {

	// Decide target format.
	format := e.Format
	if *targetFormat != "" {
		format = *targetFormat
	}

	// PbFormat.
	var (
		b   []byte
		err error
	)

	switch format {
	case npenc.PbFormat:
		d, ok := data.(proto.Message)
		if !ok {
			return fmt.Errorf("PbJsonEncoder can't encode %+v using pb format", data)
		}
		b, err = e.PbMarshalOptions.Marshal(d)
		if err != nil {
			return err
		}
		*targetFormat = npenc.PbFormat
		*targetBytes = b
		return nil

	case npenc.JsonFormat:
		switch d := data.(type) {
		case proto.Message:
			b, err = e.JsonMarshalOptions.Marshal(d)

		default:
			b, err = json.Marshal(d)
		}
		if err != nil {
			return err
		}
		*targetFormat = npenc.JsonFormat
		*targetBytes = b
		return nil

	default:
		return fmt.Errorf("PbJsonEncoder does not support format %q", format)

	}

}

// DecodeData decodes data:
//   - If src.Format == PbFormat, use protobuf.
//   - If src.Format == JsonFormat, use json.
func (e *PbJsonDecoder) DecodeData(srcFormat string, srcBytes []byte, data interface{}) error {

	switch srcFormat {
	case npenc.PbFormat:
		d, ok := data.(proto.Message)
		if !ok {
			return fmt.Errorf("PbJsonDecoder can't decode %+v using pb format", data)
		}
		return e.PbUnmarshalOptions.Unmarshal(srcBytes, d)

	case npenc.JsonFormat:
		switch d := data.(type) {
		case proto.Message:
			return e.JsonUnmarshalOptions.Unmarshal(srcBytes, d)

		default:
			return json.Unmarshal(srcBytes, data)
		}

	default:
		return fmt.Errorf("PbJsonDecoder does not support format %q", srcFormat)
	}

}
