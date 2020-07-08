package enc

import (
	"io"
)

// Encoder is used to encode/decode data.
type Encoder interface {
	// EncoderName should be a unique identity of the encoder.
	EncoderName() string

	// EncodeData encodes data to w.
	//
	// If data's type is *RawData/RawData and data.EncoderName == Encoder.EncoderName(),
	// then data.Bytes should be written to w directly without encoding.
	EncodeData(w io.Writer, data interface{}) error

	// DecodeData decodes data from r.
	//
	// If data's type is *RawData, then data.EncoderName should be set to Encoder.EncoderName(),
	// and data.Bytes should be read from r directly without decoding.
	DecodeData(r io.Reader, data interface{}) error
}

// RawData is similar to json.RawMessage which bypasses encoding/decoding.
type RawData struct {
	// EncoderName is the name of the encoder which data is encoded by.
	EncoderName string

	// Bytes is the encoded data raw bytes.
	Bytes []byte
}
