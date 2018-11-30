package nprpc

import (
	"github.com/rs/zerolog"
)

// ServerOptLogger sets logger.
func ServerOptLogger(logger *zerolog.Logger) ServerOption {
	return func(server *NatsRPCServer) error {
		if logger == nil {
			nop := zerolog.Nop()
			logger = &nop
		}
		server.logger = logger.With().Str("component", "nproto.nprpc.NatsRPCServer").Logger()
		return nil
	}
}

// ServerOptSubjectPrefix sets the subject prefix.
func ServerOptSubjectPrefix(subjPrefix string) ServerOption {
	return func(server *NatsRPCServer) error {
		server.subjectPrefix = subjPrefix
		return nil
	}
}

// ServerOptGroup sets the subscription group of the server.
func ServerOptGroup(group string) ServerOption {
	return func(server *NatsRPCServer) error {
		server.group = group
		return nil
	}
}

// ClientOptSubjectPrefix sets the subject prefix.
func ClientOptSubjectPrefix(subjPrefix string) ClientOption {
	return func(client *NatsRPCClient) error {
		client.subjectPrefix = subjPrefix
		return nil
	}
}

// ClientOptPBEncoding sets rpc encoding to protobuf.
func ClientOptPBEncoding() ClientOption {
	return func(client *NatsRPCClient) error {
		client.encoding = "pb"
		return nil
	}
}

// ClientOptJSONEncoding sets rpc encoding to json.
func ClientOptJSONEncoding() ClientOption {
	return func(client *NatsRPCClient) error {
		client.encoding = "json"
		return nil
	}
}
