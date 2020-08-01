package rpctracing

import (
	"fmt"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	. "github.com/huangjunwen/nproto/v2/rpc"
)

var (
	// ClientComponentTag is added to each client span.
	ClientComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "nprpc.client",
	}
	// ServerComponentTag is added to each server span.
	ServerComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "nprpc.server",
	}
)

var (
	// ClientOpName is used to generate operation name of a client span.
	ClientOpName = func(spec RPCSpec) string {
		return fmt.Sprintf("RPC Client %s:%s", spec.SvcName(), spec.MethodName())
	}
	// ServerOpName is used to generate operation name of a server span.
	ServerOpName = func(spec RPCSpec) string {
		return fmt.Sprintf("RPC Server %s:%s", spec.SvcName(), spec.MethodName())
	}
)
