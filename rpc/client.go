package rpc

// RPCClient is used to invoke rpc services.
type RPCClient interface {
	// MakeHandler creates a RPCHandler for a given method of a service.
	MakeHandler(spec *RPCSpec) RPCHandler
}

// RPCClientWithMWs wraps an RPCClient with RPCMiddlewares.
type RPCClientWithMWs struct {
	client RPCClient
	mws    []RPCMiddleware
}

var (
	_ RPCClient = (*RPCClientWithMWs)(nil)
)

// NewRPCClientWithMWs creates a new RPCClientWithMWs.
func NewRPCClientWithMWs(client RPCClient, mws ...RPCMiddleware) *RPCClientWithMWs {
	return &RPCClientWithMWs{
		client: client,
		mws:    mws,
	}
}

// MakeHandler implements RPCClient interface.
func (client *RPCClientWithMWs) MakeHandler(spec *RPCSpec) RPCHandler {
	handler := client.client.MakeHandler(spec)
	for i := len(client.mws) - 1; i >= 0; i-- {
		handler = client.mws[i](spec, handler)
	}
	return handler
}
