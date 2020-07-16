package nppbenc

import (
	npenc "github.com/huangjunwen/nproto/v2/enc"
)

// To converts to *npenc.RawData.
func (rawData *RawData) To() *npenc.RawData {
	return &npenc.RawData{
		Format: rawData.Format,
		Bytes:  rawData.Bytes,
	}
}

// NewRawData converts from *npenc.RawData.
func NewRawData(src *npenc.RawData) *RawData {
	return &RawData{
		Format: src.Format,
		Bytes:  src.Bytes,
	}
}
