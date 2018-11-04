package nprpc

import (
	"github.com/rs/zerolog"
)

// ServerOptSubjectPrefix sets the subject prefix.
func ServerOptSubjectPrefix(subjPrefix string) NatsRPCServerOption {
	return func(server *NatsRPCServer) error {
		server.subjPrefix = subjPrefix
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

// ServerOptLogger sets logger.
func ServerOptLogger(logger *zerolog.Logger) NatsRPCServerOption {
	return func(server *NatsRPCServer) error {
		if logger == nil {
			nop := zerolog.Nop()
			logger = &nop
		}
		server.logger = logger.With().Str("component", "nproto.nprpc.NatsRPCServer").Logger()
		return nil
	}
}

// ClientOptSubjectPrefix sets the subject prefix.
func ClientOptSubjectPrefix(subjPrefix string) NatsRPCClientOption {
	return func(client *NatsRPCClient) error {
		client.subjPrefix = subjPrefix
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
