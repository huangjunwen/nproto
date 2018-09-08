package nprpc

import (
	"context"
)

// RPCContext contains variables of current RPC invoke.
type RPCContext struct {
	SvcName string
	Method  *RPCMethod
}

// Passthru extracts passthru dict from context.
func Passthru(ctx context.Context) map[string]string {
	v := ctx.Value(passthruKey)
	if v == nil {
		return nil
	}
	return v.(map[string]string)
}

// SetPassthru merges passthru dict into context and returns a new context.
func SetPassthru(ctx context.Context, passthru map[string]string) context.Context {
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

// CurRPCCtx returns the current RPC context. It return nil if it is not called inside rpc handler.
func CurRPCCtx(ctx context.Context) *RPCContext {
	v := ctx.Value(curRPCCtxKey)
	if v == nil {
		return nil
	}
	return v.(*RPCContext)
}

// SetCurRPCCtx sets current RPC context.
func SetCurRPCCtx(ctx context.Context, rpcCtx *RPCContext) context.Context {
	return context.WithValue(ctx, curRPCCtxKey, rpcCtx)
}

type passthruKeyType struct{}

type curRPCCtxKeyType struct{}

var (
	passthruKey  = passthruKeyType{}
	curRPCCtxKey = curRPCCtxKeyType{}
)
