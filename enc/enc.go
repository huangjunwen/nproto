// Package enc contains encode related types.
package enc

// Encoder is used to encode data.
type Encoder interface {
	// EncodeData encodes data.
	//
	// If *targetFormat is not empty, then the encoder must encode data in the specified format.
	EncodeData(data interface{}, targetFormat *string, targetBytes *[]byte) error
}

// Decoder is used to decode data.
type Decoder interface {
	// DecodeData decodes data.
	DecodeData(srcFormat string, srcBytes []byte, data interface{}) error
}
