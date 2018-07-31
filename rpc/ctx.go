package rpc

import (
	"context"
)

type passthruKeyType struct{}

var (
	passthruKey = passthruKeyType{}
)

// Passthru extracts passthru dict from context.
func Passthru(ctx context.Context) map[string]string {
	v := ctx.Value(passthruKey)
	if v == nil {
		return nil
	}
	return v.(map[string]string)
}

// WithPassthru merges passthru dict into context and returns a new context.
func WithPassthru(ctx context.Context, passthru map[string]string) context.Context {
	p := map[string]string{}
	// Adds exists values first.
	for k, v := range Passthru(ctx) {
		p[k] = v
	}
	// Now adds new values.
	for k, v := range passthru {
		p[k] = v
	}
	return context.WithValue(ctx, passthruKey, p)
}
