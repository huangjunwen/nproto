package rpc

import (
	"context"
)

// RPCSpec is the contract between rpc server and client.
type RPCSpec struct {
	// SvcName is the name of service.
	SvcName string

	// MethodName is the name of method.
	MethodName string

	// NewInput is used to generate a new input parameter. Should be a pointer.
	NewInput func() interface{}

	// NewOutput is used to generate a new output parameter. Should be a pointer.
	NewOutput func() interface{}
}

// RPCHandler do the real job. RPCHandler can be client side or server side.
type RPCHandler func(context.Context, interface{}) (interface{}, error)

// RPCMiddleware wraps a RPCHandler into another one.
type RPCMiddleware func(spec *RPCSpec, handler RPCHandler) RPCHandler
