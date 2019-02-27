package nproto

import (
	"context"
	"fmt"
)

// MD is used to carry extra key/value context data (meta data) from outgoing side to
// incoming side. Each key (utf8 string) has a list of values (any bytes) associated.
// MD should be immutable once attached to context. Create a new one if you want to modify.
// (just like context.WithValue).
type MD interface {
	// Keys iterates all keys in MD.
	Keys(cb func(string) error) error

	// HasKey returns true if MD contains the specified key.
	HasKey(key string) bool

	// Values returns the list of values associated with the specified key or
	// nil if not exists. NOTE: Don't modify the returned slice and its content.
	Values(key string) [][]byte
}

// MetaData is the default implementation of MD. nil is a valid value.
type MetaData map[string][][]byte

type outgoingMDKey struct{}

type incomingMDKey struct{}

var (
	// EmptyMD is an empty MD.
	EmptyMD MD = (MetaData)(nil)
)

// NewOutgoingContextWithMD creates a new context with outgoing MD attached.
func NewOutgoingContextWithMD(ctx context.Context, md MD) context.Context {
	return context.WithValue(ctx, outgoingMDKey{}, md)
}

// NewIncomingContextWithMD creates a new context with incoming MD attached.
func NewIncomingContextWithMD(ctx context.Context, md MD) context.Context {
	return context.WithValue(ctx, incomingMDKey{}, md)
}

// MDFromOutgoingContext extracts outgoing MD from context or nil if not found.
func MDFromOutgoingContext(ctx context.Context) MD {
	v := ctx.Value(outgoingMDKey{})
	if v == nil {
		return nil
	}
	return v.(MD)
}

// MDFromIncomingCont extracts incoming MD from context or nil if not found.
func MDFromIncomingContext(ctx context.Context) MD {
	v := ctx.Value(incomingMDKey{})
	if v == nil {
		return nil
	}
	return v.(MD)
}

// NewMetaDataPairs creates a MetaData from key/value pairs. len(kv) must be even.
func NewMetaDataPairs(kv ...string) MetaData {
	if len(kv)%2 == 1 {
		panic(fmt.Errorf("NewMetaDataPairs: got odd number of kv strings"))
	}
	md := MetaData{}
	for i := 0; i < len(kv); i += 2 {
		md[kv[i]] = [][]byte{[]byte(kv[i+1])}
	}
	return md
}

// NewMetaDataFromMD creates a MetaData from MD.
func NewMetaDataFromMD(md MD) MetaData {
	if md == nil {
		return (MetaData)(nil)
	}
	ret := MetaData{}
	md.Keys(func(key string) error {
		ret[key] = md.Values(key)
		return nil
	})
	return ret
}

// Keys implements MD interface.
func (md MetaData) Keys(cb func(string) error) error {
	if len(md) == 0 {
		return nil
	}
	for key, _ := range md {
		if err := cb(key); err != nil {
			return err
		}
	}
	return nil
}

// HasKey implements MD interface.
func (md MetaData) HasKey(key string) bool {
	_, ok := md[key]
	return ok
}

// Values implements MD interface.
func (md MetaData) Values(key string) [][]byte {
	return md[key]
}
