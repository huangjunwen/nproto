package rpc

// RPCClient is used to invoke rpc services.
type RPCClient interface {
	// MakeHandler creates a RPCHandler for a method.
	MakeHandler(spec *RPCSpec) RPCHandler
}

// RPCClientFunc is an adapter to allow the use of ordinary functions as RPCClient.
type RPCClientFunc func(*RPCSpec) RPCHandler

// RPCClientWithMWs wraps an RPCClient with RPCMiddlewares.
type RPCClientWithMWs struct {
	client RPCClient
	mws    []RPCMiddleware
}

var (
	_ RPCClient = (RPCClientFunc)(nil)
	_ RPCClient = (*RPCClientWithMWs)(nil)
)

// MakeHandler implements RPCClient interface.
func (fn RPCClientFunc) MakeHandler(spec *RPCSpec) RPCHandler {
	return fn(spec)
}

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
