package rpc

// NatsRPCServerOption is option in creating NatsRPCServer.
type NatsRPCServerOption func(*NatsRPCServer) error

// NatsRPCClientOption is option in creating NatsRPCClient.
type NatsRPCClientOption func(*NatsRPCClient) error

// ServerOptNameConv sets the convertor to convert svc name to nats subject name.
func ServerOptNameConv(nameConv func(string) string) NatsRPCServerOption {
	return func(server *NatsRPCServer) error {
		server.nameConv = nameConv
		return nil
	}
}

// ServerOptGroup sets the subscription group of the server.
func ServerOptGroup(group string) NatsRPCServerOption {
	return func(server *NatsRPCServer) error {
		server.group = group
		return nil
	}
}

// ServerOptUseMiddleware adds a middleware to middleware stack.
func ServerOptUseMiddleware(mw RPCMiddleware) NatsRPCServerOption {
	return func(server *NatsRPCServer) error {
		server.mws = append(server.mws, mw)
		return nil
	}
}

// ClientOptNameConv sets the convertor to convert svc name to nats subject name.
// Should be the same as ServerOptNameConv.
func ClientOptNameConv(nameConv func(string) string) NatsRPCClientOption {
	return func(client *NatsRPCClient) error {
		client.nameConv = nameConv
		return nil
	}
}

// ClientOptPBEncoding sets rpc encoding to protobuf.
func ClientOptPBEncoding() NatsRPCClientOption {
	return func(client *NatsRPCClient) error {
		client.encoding = "pb"
		return nil
	}
}

// ClientOptJSONEncoding sets rpc encoding to json.
func ClientOptJSONEncoding() NatsRPCClientOption {
	return func(client *NatsRPCClient) error {
		client.encoding = "json"
		return nil
	}
}

// ClientOptUseMiddleware adds a middleware to middleware stack.
func ClientOptUseMiddleware(mw RPCMiddleware) NatsRPCClientOption {
	return func(client *NatsRPCClient) error {
		client.mws = append(client.mws, mw)
		return nil
	}
}
