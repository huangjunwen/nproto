package nproto

import (
	"context"
	"fmt"
)

// rpcCtx is the server side context of an RPC called. Implements context.Context interface.
type rpcCtx struct {
	context.Context
	svcName string
	method  *RPCMethod
	md      MetaData
}

// msgCtx is the subscriber side context of a message delivery. Implements context.Context interface.
type msgCtx struct {
	context.Context
	subject string
	md      MetaData
}

// MetaData is meta data.
type MetaData map[string][]string

// The following keys are for incoming context.

type rpcCtxSvcNameKey struct{}

type rpcCtxMethodKey struct{}

type rpcCtxMetaDataKey struct{}

type msgCtxSubjectKey struct{}

type msgCtxMetaDataKey struct{}

// The following keys are for outgoing context.

type outgoingMetaDataKey struct{}

// CurrRPCSvcName returns the service name of current RPC call of "" if not found.
// This function is usually called inside a server side handler.
func CurrRPCSvcName(ctx context.Context) string {
	v := ctx.Value(rpcCtxSvcNameKey{})
	if v == nil {
		return ""
	}
	return v.(string)
}

// CurrRPCMethod returns the method of current RPC call or nil if not found.
// This function is usually called inside a server side handler.
func CurrRPCMethod(ctx context.Context) *RPCMethod {
	v := ctx.Value(rpcCtxMethodKey{})
	if v == nil {
		return nil
	}
	return v.(*RPCMethod)
}

// CurrRPCMetaData returns the meta data of current RPC call or nil if not found.
// This function is usually called inside a server side handler.
func CurrRPCMetaData(ctx context.Context) MetaData {
	v := ctx.Value(rpcCtxMetaDataKey{})
	if v == nil {
		return nil
	}
	return v.(MetaData)
}

// NewRPCCtx creates a new rpc context. This function is usually used by RPCServer implementation
// to setup context for RPCHandler.
//   parent: Parent context.
//   svcName: Current rpc's service name. Use CurrRPCSvcName to get it inside RPCHandler.
//   method: Current rpc's method. Use CurrRPCMethod to get it inside RPCHandler.
//   md: Optional meta data. Use CurrRPCMetaData to get it inside RPCHandler.
func NewRPCCtx(parent context.Context, svcName string, method *RPCMethod, md MetaData) context.Context {
	return &rpcCtx{
		Context: parent,
		svcName: svcName,
		method:  method,
		md:      md,
	}
}

// Value implements context.Context interface.
func (ctx *rpcCtx) Value(key interface{}) interface{} {
	switch key.(type) {
	case rpcCtxSvcNameKey:
		return ctx.svcName
	case rpcCtxMethodKey:
		return ctx.method
	case rpcCtxMetaDataKey:
		return ctx.md
	default:
		return ctx.Context.Value(key)
	}
}

// CurrMsgSubject returns the subject of in current msg delivery or "" if not found.
// This function is usually called inside a subscriber side handler.
func CurrMsgSubject(ctx context.Context) string {
	v := ctx.Value(msgCtxSubjectKey{})
	if v == nil {
		return ""
	}
	return v.(string)
}

// CurrMsgMetaData returns the meta data of current msg delivery or nil if not found.
// This function is usually called inside a subscriber side handler.
func CurrMsgMetaData(ctx context.Context) MetaData {
	v := ctx.Value(msgCtxMetaDataKey{})
	if v == nil {
		return nil
	}
	return v.(MetaData)
}

// NewMsgCtx creates a new msg context. This function is usually used by MsgSubscriber implementation
// to setup context for MsgHandler.
//   parent: Parent context.
//   subject: Current msg's subject. Use CurrMsgSubject to get it inside MsgHandler.
//   md: Optional meta data. Use CurrMsgMetaData to get it inside MsgHandler.
func NewMsgCtx(parent context.Context, subject string, md MetaData) context.Context {
	return &msgCtx{
		Context: parent,
		subject: subject,
		md:      md,
	}
}

// Value implements context.Context interface.
func (ctx *msgCtx) Value(key interface{}) interface{} {
	switch key.(type) {
	case msgCtxSubjectKey:
		return ctx.subject
	case msgCtxMetaDataKey:
		return ctx.md
	default:
		return ctx.Context.Value(key)
	}
}

// NewOutgoingContext creates a context with meta data attached.
// This function is usually called in client side of RPC/publish side of msg delivery to send extra meta data
// to their receivers.
func NewOutgoingContext(ctx context.Context, md MetaData) context.Context {
	return context.WithValue(ctx, outgoingMetaDataKey{}, md)
}

// FromOutgoingContext extract outgoing meta data from context.
// This function is usually used by RPCClient/MsgPublisher implementations only.
func FromOutgoingContext(ctx context.Context) MetaData {
	v := ctx.Value(outgoingMetaDataKey{})
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
