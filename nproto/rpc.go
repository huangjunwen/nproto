package nproto

import (
	"context"

	"github.com/golang/protobuf/proto"
)

type RPCServer interface {
	RegistSvc(svcName string, methods map[string]RPCHandler) error

	DeregistSvc(svcName string) error

	Close() error
}

type RPCClient interface {
	InvokeSvc(svcName, methodName string) RPCHandler

	Close() error
}

type RPCHandler func(context.Context, proto.Message) (proto.Message, error)

type RPCMiddleware func(RPCHandler) RPCHandler
