package nproto

import (
	"context"
	"time"
)

type rpcCtx struct {
	svcName  string
	method   *RPCMethod
	passthru map[string]string
}

type msgCtx struct {
	subject  string
	passthru map[string]string
}

type rpcCtxSvcNameKey struct{}

type rpcCtxMethodKey struct{}

type msgCtxSubjectKey struct{}

type passthruKey struct{}

// CurrRPCSvcName returns the service name of current rpc handler or "" if not found.
// See NewRPCCtx for more detail.
func CurrRPCSvcName(ctx context.Context) string {
	v := ctx.Value(rpcCtxSvcNameKey{})
	if v == nil {
		return ""
	}
	return v.(string)
}

// CurrRPCMethod returns the method of current rpc handler or nil if not found.
// See NewRPCCtx for more detail.
func CurrRPCMethod(ctx context.Context) *RPCMethod {
	v := ctx.Value(rpcCtxMethodKey{})
	if v == nil {
		return nil
	}
	return v.(*RPCMethod)
}

// NewRPCCtx creates a new rpc context. This function is mainly used by RPCServer implementation
// to setup context for RPCHandler.
//   svcName: Current rpc's service name. Use CurrRPCSvcName to get it inside RPCHandler.
//   method: Current rpc's method. Use CurrRPCMethod to get it inside RPCHandler.
//   passthru(optional): Passthru context dict. Use Passthru to get it inside RPCHandler.
//   timeout(optional): Set the timeout of the context if > 0.
func NewRPCCtx(svcName string, method *RPCMethod, passthru map[string]string, timeout time.Duration) context.Context {
	ret := &rpcCtx{
		svcName:  svcName,
		method:   method,
		passthru: passthru,
	}
	if timeout > 0 {
		return context.WithTimeout(ret, timeout)
	}
	return ret
}

func (ctx *rpcCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (ctx *rpcCtx) Done() <-chan struct{} {
	return nil
}

func (ctx *rpcCtx) Err() error {
	return nil
}

func (ctx *rpcCtx) Value(key interface{}) interface{} {
	switch key.(type) {
	case rpcCtxSvcNameKey:
		return ctx.svcName
	case rpcCtxMethodKey:
		return ctx.method
	case passthruKey:
		return ctx.passthru
	default:
		return nil
	}
}

// CurrMsgSubject returns the subject of in current msg handler or "" if not found.
// See NewMsgCtx for more detail.
func CurrMsgSubject(ctx context.Context) string {
	v := ctx.Value(msgCtxSubjectKey{})
	if v == nil {
		return ""
	}
	return v.(string)
}

// NewMsgCtx creates a new msg context. This function is mainly used by MsgSubscriber implementation
// to setup context for MsgHandler.
//   subject: Current msg's subject. Use CurrMsgSubjec to get it inside MsgHandler.
//   passthru(optional): Passthru context dict. Use Passthru to get it inside MsgHandler.
func NewMsgCtx(subject string, passthru map[string]string) context.Context {
	return &msgCtx{
		subject:  subject,
		passthru: passthru,
	}
}

func (ctx *msgCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (ctx *msgCtx) Done() <-chan struct{} {
	return nil
}

func (ctx *msgCtx) Err() error {
	return nil
}

func (ctx *msgCtx) Value(key interface{}) interface{} {
	switch key.(type) {
	case msgCtxSubjectKey:
		return ctx.subject
	case passthruKey:
		return ctx.passthru
	default:
		return nil
	}
}

// Passthru extracts passthru dict from context. This dict is used to pass context values. (e.g. trace information)
// NOTE: It is not used to pass optional params.
func Passthru(ctx context.Context) map[string]string {
	v := ctx.Value(passthruKey{})
	if v == nil {
		return nil
	}
	return v.(map[string]string)
}

// AddPassthru adds a new kv to passthru dict and returns a new context.
// NOTE: the original dict in original context is not modified.
func AddPassthru(ctx context.Context, k, v string) context.Context {
	p := map[string]string{}
	// Adds exists values first.
	for a, b := range Passthru(ctx) {
		p[a] = b
	}
	// Now adds new value.
	p[k] = v
	return context.WithValue(ctx, passthruKey{}, p)
}

// AddPassthru adds key-values to passthru dict and returns a new context.
// NOTE: the original dict in original context is not modified.
func AddPassthruDict(ctx context.Context, dict map[string]string) context.Context {
	p := map[string]string{}
	// Adds exists values first.
	for a, b := range Passthru(ctx) {
		p[a] = b
	}
	// Now adds new values.
	for a, b := range dict {
		p[a] = b
	}
	return context.WithValue(ctx, passthruKey{}, p)
}
