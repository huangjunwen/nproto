package nproto

import (
	"context"

	"github.com/golang/protobuf/proto"
)

// RPCServer is used to serve rpc services.
type RPCServer interface {
	// RegistSvc regist a service with given method set and associated handlers.
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
	// NewInput is used to generate a new input message.
	NewInput func() proto.Message
	// NewOutput is used to generate a new output message.
	NewOutput func() proto.Message
}

// RPCHandler do the real job. RPCHandler can be client side or server side.
// MetaData attached to `ctx` must be passed unmodified from client side to
// its corresponding server side.
type RPCHandler func(context.Context, proto.Message) (proto.Message, error)
