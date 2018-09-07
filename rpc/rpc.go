package nprpc

import (
	"context"

	"github.com/golang/protobuf/proto"
)

// RPCServer is the server side of service.
type RPCServer interface {
	// RegistSvc regist a service with given method set and associated handlers.
	RegistSvc(svcName string, methods map[*RPCMethod]RPCHandler) error

	// DeregistSvc deregist a service.
	DeregistSvc(svcName string) error

	// Close the server.
	Close() error
}

// RPCClient is the client side of service.
type RPCClient interface {
	// MakeHandler creates a RPCHandler for a given method of a service.
	MakeHandler(svcName string, method *RPCMethod) RPCHandler

	// Close the client.
	Close() error
}

// RPCMethod contains meta and type information of a given method.
type RPCMethod struct {
	// Name is the name of this method.
	Name string
	// NewInput is used to generate a new input message.
	NewInput func() proto.Message
	// NewOutput is used to generate a new output message.
	NewOutput func() proto.Message
}

// RPCHandler is where real logic resides.
type RPCHandler func(context.Context, proto.Message) (proto.Message, error)

// RPCMiddleware is used to decorate RPCHandler.
type RPCMiddleware func(RPCHandler) RPCHandler

type decRPCServer struct {
	RPCServer
	mws []RPCMiddleware
}

type decRPCClient struct {
	RPCClient
	mws []RPCMiddleware
}

// DecorateRPCServer decorates a RPCServer with RPCMiddlewares.
func DecorateRPCServer(server RPCServer, mws ...RPCMiddleware) RPCServer {
	return &decRPCServer{
		RPCServer: server,
		mws:       mws,
	}
}

// RegistSvc implements RPCServer interface.
func (server *decRPCServer) RegistSvc(svcName string, methods map[*RPCMethod]RPCHandler) error {
	// Decorate handler.
	ms := make(map[*RPCMethod]RPCHandler)
	for method, handler := range methods {
		for i := len(server.mws) - 1; i >= 0; i-- {
			handler = server.mws[i](handler)
		}
		ms[method] = handler
	}

	return server.RPCServer.RegistSvc(svcName, ms)
}

// DecorateRPCClient decorates a RPCClient with RPCMiddlewares.
func DecorateRPCClient(client RPCClient, mws ...RPCMiddleware) RPCClient {
	return &decRPCClient{
		RPCClient: client,
		mws:       mws,
	}
}

// MakeHandler implements RPCClient interface.
func (client *decRPCClient) MakeHandler(svcName string, method *RPCMethod) RPCHandler {
	handler := client.RPCClient.MakeHandler(svcName, method)

	// Decorate handler.
	for i := len(client.mws) - 1; i >= 0; i-- {
		handler = client.mws[i](handler)
	}
	return handler
}
