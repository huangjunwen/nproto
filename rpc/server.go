package rpc

// RPCServer is used to serve rpc services.
type RPCServer interface {
	// RegistHandler regists a handler if not nil, or deregist it otherwise.
	RegistHandler(spec RPCSpec, handler RPCHandler) error
}

// RPCServerFunc is an adapter to allow the use of ordinary functions as RPCServer.
type RPCServerFunc func(RPCSpec, RPCHandler) error

// RPCServerWithMWs wraps an RPCServer with RPCMiddlewares.
type RPCServerWithMWs struct {
	server RPCServer
	mws    []RPCMiddleware
}

var (
	_ RPCServer = (RPCServerFunc)(nil)
	_ RPCServer = (*RPCServerWithMWs)(nil)
)

// RegistHandler implements RPCServer interface.
func (fn RPCServerFunc) RegistHandler(spec RPCSpec, handler RPCHandler) error {
	return fn(spec, handler)
}

// NewRPCServerWithMWs creates a new RPCServerWithMWs.
func NewRPCServerWithMWs(server RPCServer, mws ...RPCMiddleware) *RPCServerWithMWs {
	return &RPCServerWithMWs{
		server: server,
		mws:    mws,
	}
}

// RegistHandler implements RPCServer interface.
func (server *RPCServerWithMWs) RegistHandler(spec RPCSpec, handler RPCHandler) error {
	n := len(server.mws)
	for i := n - 1; i >= 0; i-- {
		handler = server.mws[i](spec, handler)
	}
	return server.server.RegistHandler(spec, handler)
}
