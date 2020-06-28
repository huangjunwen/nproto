package md

import (
	"context"
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

type outgoingMDKey struct{}

type incomingMDKey struct{}

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
