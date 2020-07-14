package nppbenc

import (
	npenc "github.com/huangjunwen/nproto/v2/enc"
)

// To writes to *npenc.RawData.
func (rawData *RawData) To() *npenc.RawData {
	return &npenc.RawData{
		EncoderName: rawData.EncoderName,
		Bytes:       rawData.Bytes,
	}
}

// From reads from src.
func (rawData *RawData) From(src *npenc.RawData) {
	rawData.EncoderName = src.EncoderName
	rawData.Bytes = src.Bytes
}
