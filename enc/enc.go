// Package enc contains encode related types.
package enc

import (
	"fmt"
)

// Encoder is used to encode data.
type Encoder interface {
	// EncodeData encodes data to target.
	//
	// If target.Format is not empty, then the encoder must encode data in the specified format.
	EncodeData(data interface{}, target *RawData) error
}

// Decoder is used to decode data.
type Decoder interface {
	// DecodeData decodes data from src.
	DecodeData(src *RawData, data interface{}) error
}

// RawData is encoded data. Similar to json.RawMessage which bypasses encoding/decoding.
type RawData struct {
	// Format is the wire format of Bytes. (e.g. "json")
	Format string

	// Bytes is the encoded raw bytes.
	Bytes []byte
}

type rawDataEncoder struct {
	encoder Encoder
}

type rawDataDecoder struct {
	decoder Decoder
}

// NewEncoder wraps an encoder to add RawData awareness: If data is *RawData and
// its format satisfies target's format requirement, then it is copied directly to target
// without encoding.
//
// Otherwise passthrough to encoder.EncodeData if it's not nil.
func NewEncoder(encoder Encoder) Encoder {
	return &rawDataEncoder{encoder}
}

// NewDecoder wraps a decoder to add RawData awareness: If data is *RawData,
// then it is copied directly from src without decoding.
//
// Otherwise passthrough to decoder.DecodeData if it's not nil.
func NewDecoder(decoder Decoder) Decoder {
	return &rawDataDecoder{decoder}
}

func (e *rawDataEncoder) EncodeData(data interface{}, target *RawData) error {

	switch src := data.(type) {
	case *RawData:
		if target.Format != "" && src.Format != target.Format {
			return fmt.Errorf("Target requires format %q but src RawData format is %q", target.Format, src.Format)
		}
		*target = *src
		return nil

	default:
		if e.encoder == nil {
			return fmt.Errorf("No encoder to encode %+v", data)
		}
		return e.encoder.EncodeData(data, target)
	}

}

func (e *rawDataDecoder) DecodeData(src *RawData, data interface{}) error {

	switch target := data.(type) {
	case *RawData:
		*target = *src
		return nil

	default:
		if e.decoder == nil {
			return fmt.Errorf("No decoder to decode %+v", data)
		}
		return e.decoder.DecodeData(src, data)
	}

}
