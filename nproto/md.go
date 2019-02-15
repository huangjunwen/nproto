package nproto

import (
	"context"
	"fmt"
)

// MetaData is used to carry extra context data from client/producer (outgoing) side
// to server/consumer (incoming) side.
type MetaData map[string][]string

type incomingMDKey struct{}

type outgoingMDKey struct{}

// NewIncomingContextWithMD creates a new context with incoming MetaData attached.
// This function is usually used by server/consumer side to setup context for handlers.
func NewIncomingContextWithMD(ctx context.Context, md MetaData) context.Context {
	return context.WithValue(ctx, incomingMDKey{}, md)
}

// NewOutgoingContextWithMD creates a new context with outgoing MetaData attached.
// This function is usually used by client/producer side to attach MetaData.
func NewOutgoingContextWithMD(ctx context.Context, md MetaData) context.Context {
	return context.WithValue(ctx, outgoingMDKey{}, md)
}

// MDFromOutgoingContext extracts outgoing MetaData from context.
func MDFromOutgoingContext(ctx context.Context) MetaData {
	v := ctx.Value(outgoingMDKey{})
	if v == nil {
		return nil
	}
	return v.(MetaData)
}

// MDFromIncomingCont extracts incoming MetaData from context.
func MDFromIncomingContext(ctx context.Context) MetaData {
	v := ctx.Value(incomingMDKey{})
	if v == nil {
		return nil
	}
	return v.(MetaData)
}

// NewMetaDataPairs creates a MetaData from key/value pairs. len(kv) must be even.
func NewMetaDataPairs(kv ...string) MetaData {
	if len(kv)%2 == 1 {
		panic(fmt.Errorf("NewMetaDataPairs: got odd number of kv strings"))
	}
	md := MetaData{}
	for i := 0; i < len(kv); i += 2 {
		md[kv[i]] = []string{kv[i+1]}
	}
	return md
}

// Get gets the first value of the given key or "" if not found.
func (md MetaData) Get(key string) string {
	vals := md[key]
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Set sets the values of the given key. Old values of the key are overwritten.
func (md MetaData) Set(key string, vals ...string) {
	md[key] = vals
}

// Append appends values to the given key. Old values of the key are preserved.
func (md MetaData) Append(key string, vals ...string) {
	md[key] = append(md[key], vals...)
}

// Len returns the length of the map.
func (md MetaData) Len() int {
	return len(md)
}

// Copy returns a copy of MetaData.
func (md MetaData) Copy() MetaData {
	return Join(md)
}

// Join joins a number of MetaDatas into one.
func Join(mds ...MetaData) MetaData {
	ret := MetaData{}
	for _, md := range mds {
		for key, vals := range md {
			ret[key] = append(ret[key], vals...)
		}
	}
	return ret
}
