package enc

import (
	"io"
)

// Encoder is used to encode/decode payload.
type Encoder interface {
	// Name should be a unique identity of encoder.
	Name() string

	// EncodePayload encodes payload to w.
	//
	// If payload's type is *RawPayload/RawPayload and payload.EncoderName == Encoder.Name(),
	// then payload.Data should be written to w directly without encoding.
	EncodePayload(w io.Writer, payload interface{}) error

	// DecodePayload decodes payload from r.
	//
	// If payload's type is *RawPayload, then payload.EncoderName should be set to Encoder.Name(),
	// and payload.Data should be read from r directly without decoding.
	DecodePayload(r io.Reader, payload interface{}) error
}

// RawPayload is similar to json.RawMessage which bypasses encoding/decoding.
type RawPayload struct {
	// EncoderName is the name of the encoder which payload is encoded by.
	EncoderName string

	// Data is the encoded payload raw bytes.
	Data []byte
}
