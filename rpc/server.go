package rpc

// RPCServer is used to serve rpc services.
type RPCServer interface {
	// RegistHandler regists a handler for a given method of a service.
	RegistHandler(spec *RPCSpec, handler RPCHandler) error
}

// RPCServerWithMWs wraps an RPCServer with RPCMiddlewares.
type RPCServerWithMWs struct {
	server RPCServer
	mws    []RPCMiddleware
}

var (
	_ RPCServer = (*RPCServerWithMWs)(nil)
)

// NewRPCServerWithMWs creates a new RPCServerWithMWs.
func NewRPCServerWithMWs(server RPCServer, mws ...RPCMiddleware) *RPCServerWithMWs {
	return &RPCServerWithMWs{
		server: server,
		mws:    mws,
	}
}

// RegistHandler implements RPCServer interface.
func (server *RPCServerWithMWs) RegistHandler(spec *RPCSpec, handler RPCHandler) error {
	n := len(server.mws)
	for i := n - 1; i >= 0; i-- {
		handler = server.mws[i](spec, handler)
	}
	return server.server.RegistHandler(spec, handler)
}
