package nppbenc

import (
	npenc "github.com/huangjunwen/nproto/v2/enc"
)

// To writes to *npenc.RawData.
func (rawData *RawData) To() *npenc.RawData {
	return &npenc.RawData{
		Format: rawData.Format,
		Bytes:  rawData.Bytes,
	}
}

// From reads from src.
func (rawData *RawData) From(src *npenc.RawData) {
	rawData.Format = src.Format
	rawData.Bytes = src.Bytes
}
