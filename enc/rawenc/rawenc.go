// Package rawenc contains Encoder/Decoder bypasses encoding/decoding.
package rawenc

import (
	"fmt"

	npenc "github.com/huangjunwen/nproto/v2/enc"
)

// RawData is encoded data. Similar to json.RawMessage which bypasses encoding/decoding.
type RawData struct {
	// Format is the wire format of Bytes. (e.g. "json")
	Format string

	// Bytes is the encoded raw bytes.
	Bytes []byte
}

type rawDataEncoder struct {
	encoder npenc.Encoder
}

type rawDataDecoder struct {
	decoder npenc.Decoder
}

var (
	// DefaultRawEncoder can encodes *RawData only.
	DefaultRawEncoder npenc.Encoder = NewEncoder(nil)

	// DefaultRawDecoder can decodes *RawData only.
	DefaultRawDecoder npenc.Decoder = NewDecoder(nil)
)

// NewEncoder wraps an encoder to add RawData awareness: If data is *RawData and
// its format satisfies targetFormat's requirement, then it is copied directly to
// targetForma/targetBytes without encoding.
//
// Otherwise passthrough to encoder.EncodeData if it's not nil.
func NewEncoder(encoder npenc.Encoder) npenc.Encoder {
	return &rawDataEncoder{encoder}
}

// NewDecoder wraps a decoder to add RawData awareness: If data is *RawData,
// then it is copied directly from srcFormat/srcBytes without decoding.
//
// Otherwise passthrough to decoder.DecodeData if it's not nil.
func NewDecoder(decoder npenc.Decoder) npenc.Decoder {
	return &rawDataDecoder{decoder}
}

func (e *rawDataEncoder) EncodeData(data interface{}, targetFormat *string, targetBytes *[]byte) error {

	switch src := data.(type) {
	case *RawData:
		if *targetFormat != "" && src.Format != *targetFormat {
			return fmt.Errorf("TargetFormat is %q but src RawData format is %q", *targetFormat, src.Format)
		}
		*targetFormat = src.Format
		*targetBytes = src.Bytes
		return nil

	default:
		if e.encoder == nil {
			return fmt.Errorf("No encoder to encode %+v", data)
		}
		return e.encoder.EncodeData(data, targetFormat, targetBytes)
	}

}

func (e *rawDataDecoder) DecodeData(srcFormat string, srcBytes []byte, data interface{}) error {

	switch target := data.(type) {
	case *RawData:
		target.Format = srcFormat
		target.Bytes = srcBytes
		return nil

	default:
		if e.decoder == nil {
			return fmt.Errorf("No decoder to decode %+v", data)
		}
		return e.decoder.DecodeData(srcFormat, srcBytes, data)
	}

}
