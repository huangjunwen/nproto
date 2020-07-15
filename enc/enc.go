// Package enc contains encode related types.
package enc

import (
	"fmt"
)

// Encoder is used to encode/decode data.
type Encoder interface {
	// EncoderName is used to match Encoder for encoding/decoding.
	EncoderName() string

	// EncodeData encodes data to w.
	EncodeData(data interface{}, w *[]byte) error

	// DecodeData decodes data from r.
	DecodeData(r []byte, data interface{}) error
}

// RawData is encoded data. Similar to json.RawMessage which bypasses encoding/decoding.
type RawData struct {
	// EncoderName is encoder's name which data is encoded by.
	EncoderName string

	// Bytes is the encoded raw bytes.
	Bytes []byte
}

type rawDataEncoder struct {
	Encoder
}

// NewEncoder wraps an encoder to add RawData awareness:
//   - EncodeData: If data is *RawData/RawData and EncoderName is the same as the encoder, then it is copied directly to w without encoding.
//   - DecodeData: If data is *RawData, then it is copied directly from r without decoding.
func NewEncoder(encoder Encoder) Encoder {
	return &rawDataEncoder{encoder}
}

func (e *rawDataEncoder) EncodeData(data interface{}, w *[]byte) error {

	var rawData *RawData
	switch d := data.(type) {
	case *RawData:
		rawData = d

	case RawData:
		rawData = &d

	default:
		return e.Encoder.EncodeData(d, w)
	}

	if rawData.EncoderName != e.EncoderName() {
		return fmt.Errorf("Encoder<%s> can't encode RawData<%s>", e.EncoderName(), rawData.EncoderName)
	}
	*w = rawData.Bytes
	return nil

}

func (e *rawDataEncoder) DecodeData(r []byte, data interface{}) error {

	switch d := data.(type) {
	case *RawData:
		d.EncoderName = e.EncoderName()
		d.Bytes = r
		return nil

	default:
		return e.Encoder.DecodeData(r, d)
	}
}
