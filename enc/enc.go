package enc

import (
	"io"
)

// Encoder is used to encode/decode data.
type Encoder interface {
	// Name should be a unique identity of encoder.
	Name() string

	// EncodeData encodes data to w.
	//
	// If data's type is *RawData/RawData and data.EncoderName == Encoder.Name(),
	// then data.Data should be written to w directly without encoding.
	EncodeData(w io.Writer, data interface{}) error

	// DecodeData decodes data from r.
	//
	// If data's type is *RawData, then data.EncoderName should be set to Encoder.Name(),
	// and data.Data should be read from r directly without decoding.
	DecodeData(r io.Reader, data interface{}) error
}

// RawData is similar to json.RawMessage which bypasses encoding/decoding.
type RawData struct {
	// EncoderName is the name of the encoder which data is encoded by.
	EncoderName string

	// Bytes is the encoded data raw bytes.
	Bytes []byte
}
