package jsonenc

import (
	"io"
)

// JsonIOMarshaler is similar to json.Marshaler, but write to a io.Writer.
type JsonIOMarshaler interface {
	// MarshalJSONToWriter write marshaled json data to a writer.
	MarshalJSONToWriter(io.Writer) error
}

// JsonIOUnmarshaler is similar to json.Unmarshaler, but read from a io.Reader.
type JsonIOUnmarshaler interface {
	// UnmarshalJSONFromReader read json data from a reader.
	UnmarshalJSONFromReader(io.Reader) error
}
