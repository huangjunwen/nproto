package rpc

import (
	"context"
)

// RPCServer is used to serve rpc services.
type RPCServer interface {
	// RegistSvc regist a service with given a set of methods and their associated handlers.
	RegistSvc(svcName string, methods map[*RPCMethod]RPCHandler) error

	// DeregistSvc deregist a service.
	DeregistSvc(svcName string) error
}

// RPCClient is used to invoke rpc services.
type RPCClient interface {
	// MakeHandler creates a RPCHandler for a given method of a service.
	MakeHandler(svcName string, method *RPCMethod) RPCHandler
}

// RPCMethod contains meta information of a given method.
type RPCMethod struct {
	// Name is the name of this method.
	Name string

	// NewInput is used to generate a new input parameter.
	NewInput func() interface{}

	// NewOutput is used to generate a new output parameter.
	NewOutput func() interface{}
}

// RPCHandler do the real job. RPCHandler can be client side or server side.
type RPCHandler func(context.Context, interface{}) (interface{}, error)

// RPCMiddleware wraps an RPCHandler into another one. The params are (svcName, method, handler).
type RPCMiddleware func(string, *RPCMethod, RPCHandler) RPCHandler
