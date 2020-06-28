package rpc

// RPCServerWithMWs wraps an RPCServer with RPCMiddlewares.
type RPCServerWithMWs struct {
	server RPCServer
	mws    []RPCMiddleware
}

// RPCClientWithMWs wraps an RPCClient with RPCMiddlewares.
type RPCClientWithMWs struct {
	client RPCClient
	mws    []RPCMiddleware
}

var (
	_ RPCServer = (*RPCServerWithMWs)(nil)
	_ RPCClient = (*RPCClientWithMWs)(nil)
)

// NewRPCServerWithMWs creates a new RPCServerWithMWs.
func NewRPCServerWithMWs(server RPCServer, mws ...RPCMiddleware) *RPCServerWithMWs {
	return &RPCServerWithMWs{
		server: server,
		mws:    mws,
	}
}

// RegistSvc implements RPCServer interface. Which will wrap methods with middlewares.
func (server *RPCServerWithMWs) RegistSvc(svcName string, methods map[*RPCMethod]RPCHandler) error {
	n := len(server.mws)
	methods2 := make(map[*RPCMethod]RPCHandler)
	for method, handler := range methods {
		for i := n - 1; i >= 0; i-- {
			handler = server.mws[i](svcName, method, handler)
		}
		methods2[method] = handler
	}
	return server.server.RegistSvc(svcName, methods2)
}

// DeregistSvc implements RPCServer interface.
func (server *RPCServerWithMWs) DeregistSvc(svcName string) error {
	return server.server.DeregistSvc(svcName)
}

// NewRPCClientWithMWs creates a new RPCClientWithMWs.
func NewRPCClientWithMWs(client RPCClient, mws ...RPCMiddleware) *RPCClientWithMWs {
	return &RPCClientWithMWs{
		client: client,
		mws:    mws,
	}
}

// MakeHandler implements RPCClient interface.
func (client *RPCClientWithMWs) MakeHandler(svcName string, method *RPCMethod) RPCHandler {
	handler := client.client.MakeHandler(svcName, method)
	for i := len(client.mws) - 1; i >= 0; i-- {
		handler = client.mws[i](svcName, method, handler)
	}
	return handler
}
