package rpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"

	"github.com/huangjunwen/nproto/rpc/enc"
)

var (
	// Default convertor for service name and nats subject name.
	DefaultSvcSubjNameConvertor = func(name string) string { return fmt.Sprint("svc.%s", name) }
	// Default queue group.
	DefaultGroup = "def"
)

var (
	ErrServerClosed = errors.New("NatsRPCServer: server closed")
	ErrDupSvcName   = errors.New("NatsRPCServer: duplicated svc name")
)

type NatsRPCServer struct {
	// Options
	nameConv func(string) string // svcName -> subjName
	group    string              // server group
	mws      []RPCMiddleware     // middleware

	mu   sync.RWMutex
	conn NatsConn                      // nil if closed
	subs map[string]*nats.Subscription // svcName -> Subscription
}

type NatsRPCServerOption func(*NatsRPCServer) error

func NewNatsRPCServer(conn NatsConn, opts ...NatsRPCServerOption) (*NatsRPCServer, error) {
	server := &NatsRPCServer{
		nameConv: DefaultSvcSubjNameConvertor,
		group:    DefaultGroup,
		conn:     conn,
		subs:     make(map[string]*nats.Subscription),
	}
	for _, opt := range opts {
		if err := opt(server); err != nil {
			return nil, err
		}
	}

	return server, nil
}

func (server *NatsRPCServer) RegistSvc(svcName string, methods map[string]*RPCMethod, handlers map[string]RPCHandler) error {
	hs := make(map[string]RPCHandler)
	// Decorate handlers with middlewares.
	for methodName, handler := range handlers {
		for i := len(server.mws) - 1; i >= 0; i-- {
			handler = server.mws[i](handler)
		}
		hs[methodName] = handler
	}

	// First (read) lock: get conn and some checks.
	server.mu.RLock()
	conn := server.conn
	sub := server.subs[svcName]
	server.mu.RUnlock()
	if conn == nil {
		return ErrServerClosed
	}
	if sub != nil {
		return ErrDupSvcName
	}

	// Subscribe.
	sub2, err := server.conn.QueueSubscribe(
		server.nameConv(svcName)+".>",
		server.group,
		server.msgHandler(svcName, methods, hs),
	)
	if err != nil {
		return err
	}

	// Second (write) lock: set subscription
	server.mu.Lock()
	conn = server.conn
	sub = server.subs[svcName]
	if conn != nil && sub == nil {
		server.subs[svcName] = sub2
	}
	server.mu.Unlock()
	if conn == nil {
		sub2.Unsubscribe()
		return ErrServerClosed
	}
	if sub != nil {
		sub2.Unsubscribe()
		return ErrDupSvcName
	}

	return nil
}

func (server *NatsRPCServer) DeregistSvc(svcName string) error {
	// Pop svcName.
	server.mu.Lock()
	sub := server.subs[svcName]
	delete(server.subs, svcName)
	server.mu.Unlock()

	// Unsubscribe.
	if sub != nil {
		return sub.Unsubscribe()
	}
	return nil
}

func (server *NatsRPCServer) Close() error {
	// Set conn to nil to indicate close.
	server.mu.Lock()
	conn := server.conn
	subs := server.subs
	server.conn = nil
	server.subs = nil
	server.mu.Unlock()

	// Release subscriptions.
	if len(subs) != 0 {
		for _, sub := range subs {
			sub.Unsubscribe()
		}
	}

	// Multiple calls to Close.
	if conn == nil {
		return ErrServerClosed
	}
	return nil
}

func (server *NatsRPCServer) msgHandler(svcName string, methods map[string]*RPCMethod, handlers map[string]RPCHandler) nats.MsgHandler {
	// Some checks.
	for methodName, method := range methods {
		if method.Name != methodName {
			panic(fmt.Errorf("NatsRPCServer: method name mismatch %+q != %+q", method.Name, methodName))
		}
		if handlers[methodName] == nil {
			panic(fmt.Errorf("NatsRPCServer: handler not found for method %+q", methodName))
		}
	}

	prefix := server.nameConv(svcName) + "."
	return func(msg *nats.Msg) {
		go func() {
			// Subject should be in the form of "subj.enc.method".
			// Extract encoding and method from it.
			if !strings.HasPrefix(msg.Subject, prefix) {
				// TODO: log, this should not happen.
				return
			}
			parts := strings.Split(msg.Subject[len(prefix):], ".")
			if len(parts) != 2 {
				// Ignore.
				return
			}
			encoding, methodName := parts[0], parts[1]

			// Check encoding.
			switch encoding {
			case "pb", "json":
			default:
				// Ignore.
				return
			}

			// Check method.
			method := methods[methodName]
			handler := handlers[methodName]
			if method == nil {
				server.replyError(msg.Reply, fmt.Errorf("Method %+q not found", methodName), encoding)
				return
			}

			// Parse request payload.
			req := &enc.RPCRequest{
				Param: method.NewInput(),
			}
			if err := server.chooseEncoder(encoding).DecodeRequest(msg.Data, req); err != nil {
				server.replyError(msg.Reply, err, encoding)
				return
			}

			// Setup context.
			ctx := context.Background()
			if req.Timeout != nil {
				ctx, _ = context.WithTimeout(ctx, *req.Timeout)
			}
			if len(req.Passthru) != 0 {
				ctx = WithPassthru(ctx, req.Passthru)
			}

			// Handle.
			output, err := handler(ctx, req.Param)
			if err != nil {
				server.replyError(msg.Reply, err, encoding)
			} else {
				server.replyResult(msg.Reply, output, encoding)
			}

		}()
	}
	return nil
}

func (server *NatsRPCServer) chooseEncoder(encoding string) enc.RPCServerEncoder {
	switch encoding {
	case "json":
		return enc.JSONServerEncoder{}
	default:
		return enc.PBServerEncoder{}
	}
}

func (server *NatsRPCServer) reply(subj string, r *enc.RPCReply, encoding string) {
	// Encode reply.
	data, err := server.chooseEncoder(encoding).EncodeReply(r)
	if err != nil {
		goto Err
	}

	// Publish reply.
	err = server.conn.PublishRequest(subj, "", data)
	if err != nil {
		goto Err
	}
	return

Err:
	// TODO: log error
	return
}

func (server *NatsRPCServer) replyResult(subj string, result proto.Message, encoding string) {
	server.reply(subj, &enc.RPCReply{
		Result: result,
	}, encoding)
}

func (server *NatsRPCServer) replyError(subj string, err error, encoding string) {
	server.reply(subj, &enc.RPCReply{
		Error: err,
	}, encoding)
}
