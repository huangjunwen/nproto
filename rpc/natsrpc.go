package rpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/rs/zerolog"

	"github.com/huangjunwen/nproto/rpc/enc"
)

var (
	// Default convertor for service name and nats subject name.
	DefaultSvcSubjNameConvertor = func(name string) string { return fmt.Sprintf("svc.%s", name) }
	// Default queue group.
	DefaultGroup = "def"
	// Default RPC encoding.
	DefaultEncoding = "pb"
)

var (
	ErrNatsConnMaxRecon = errors.New("natsrpc: nats.Conn should have MaxReconnect < 0")
	ErrServerClosed     = errors.New("natsrpc: server closed")
	ErrClientClosed     = errors.New("natsrpc: client closed")
)

// NatsRPCServer implements RPCServer.
type NatsRPCServer struct {
	// Options
	nameConv func(string) string // svcName -> subjName
	group    string              // server group
	mws      []RPCMiddleware     // middlewares
	logger   zerolog.Logger      // logger

	mu   sync.RWMutex
	conn *nats.Conn                    // nil if closed
	subs map[string]*nats.Subscription // svcName -> Subscription
}

// NatsRPCClient implements RPCClient.
type NatsRPCClient struct {
	// Options
	nameConv func(string) string // svcName -> subjName
	encoding string              // rpc encoding
	mws      []RPCMiddleware     // middlewares

	mu   sync.RWMutex
	conn *nats.Conn
}

var (
	_ RPCServer = (*NatsRPCServer)(nil)
	_ RPCClient = (*NatsRPCClient)(nil)
)

// NewNatsRPCServer creates a new NatsRPCServer. `conn` should be a long-lived nats connection.
// e.g. Always reconnect.
func NewNatsRPCServer(conn *nats.Conn, opts ...NatsRPCServerOption) (*NatsRPCServer, error) {
	if conn.Opts.MaxReconnect >= 0 {
		return nil, ErrNatsConnMaxRecon
	}
	server := &NatsRPCServer{
		nameConv: DefaultSvcSubjNameConvertor,
		group:    DefaultGroup,
		logger:   zerolog.Nop(),
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

// RegistSvc implements RPCServer interface.
func (server *NatsRPCServer) RegistSvc(svcName string, methods map[*RPCMethod]RPCHandler) error {
	// Decorate rpc handlers with middlewares.
	ms := make(map[*RPCMethod]RPCHandler)
	for method, handler := range methods {
		for i := len(server.mws) - 1; i >= 0; i-- {
			handler = server.mws[i](handler)
		}
		ms[method] = handler
	}

	// Create msg handler.
	h, err := server.msgHandler(svcName, ms)
	if err != nil {
		return err
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
		return fmt.Errorf("natsrpc: duplicated service name %+q", svcName)
	}

	// Subscribe.
	sub2, err := server.conn.QueueSubscribe(
		server.nameConv(svcName)+".>",
		server.group,
		h,
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
		return fmt.Errorf("natsrpc: duplicated service name %+q", svcName)
	}

	return nil
}

// DeregistSvc implements RPCServer interface.
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

// Close implements RPCServer interface.
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

func (server *NatsRPCServer) msgHandler(svcName string, methods map[*RPCMethod]RPCHandler) (nats.MsgHandler, error) {
	// Method name -> method
	methodNames := make(map[string]*RPCMethod)
	for method, _ := range methods {
		if _, found := methodNames[method.Name]; found {
			return nil, fmt.Errorf("natsrpc: duplicated method name %+q", method.Name)
		}
		methodNames[method.Name] = method
	}

	// Subject prefix.
	prefix := server.nameConv(svcName) + "."

	return func(msg *nats.Msg) {
		go func() {
			// Subject should be in the form of "subj.enc.method".
			// Extract encoding and method from it.
			if !strings.HasPrefix(msg.Subject, prefix) {
				server.logger.Error().Err(fmt.Errorf("Unexpected msg with subject: %+q", msg.Subject)).Msg("")
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
			method, found := methodNames[methodName]
			if !found {
				server.replyError(msg.Reply, fmt.Errorf("natsrpc: method %+q not found", methodName), encoding)
				return
			}
			handler := methods[method]

			// Parse request payload.
			req := &enc.RPCRequest{
				Param: method.NewInput(),
			}
			if err := chooseServerEncoder(encoding).DecodeRequest(msg.Data, req); err != nil {
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
			result, err := handler(ctx, req.Param)
			if err != nil {
				server.replyError(msg.Reply, err, encoding)
			} else {
				server.replyResult(msg.Reply, result, encoding)
			}

		}()
	}, nil
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

func (server *NatsRPCServer) reply(subj string, r *enc.RPCReply, encoding string) {
	// Encode reply.
	data, err := chooseServerEncoder(encoding).EncodeReply(r)
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
	server.logger.Error().Err(err).Msg("")
	return
}

// NewNatsRPCClient creates a new NatsRPCClient. `conn` should be a long-lived nats connection.
// e.g. Always reconnect.
func NewNatsRPCClient(conn *nats.Conn, opts ...NatsRPCClientOption) (*NatsRPCClient, error) {
	if conn.Opts.MaxReconnect >= 0 {
		return nil, ErrNatsConnMaxRecon
	}
	client := &NatsRPCClient{
		nameConv: DefaultSvcSubjNameConvertor,
		encoding: DefaultEncoding,
		conn:     conn,
	}
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// MakeHandler implements RPCClient interface.
func (client *NatsRPCClient) MakeHandler(svcName string, method *RPCMethod) RPCHandler {

	encoder := chooseClientEncoder(client.encoding)
	subj := fmt.Sprintf("%s.%s.%s", client.nameConv(svcName), client.encoding, method.Name)
	handler := func(ctx context.Context, input proto.Message) (proto.Message, error) {

		// Get conn and check closed.
		client.mu.RLock()
		conn := client.conn
		client.mu.RUnlock()
		if conn == nil {
			return nil, ErrClientClosed
		}

		// Construct request.
		req := &enc.RPCRequest{
			Param: input,
		}
		if dl, ok := ctx.Deadline(); ok {
			dur := dl.Sub(time.Now())
			if dur <= 0 {
				return nil, context.DeadlineExceeded
			} else {
				req.Timeout = &dur
			}
		}
		passthru := Passthru(ctx)
		if len(passthru) > 0 {
			req.Passthru = passthru
		}

		// Encode request.
		data, err := encoder.EncodeRequest(req)
		if err != nil {
			return nil, err
		}

		// Send request.
		msg, err := conn.RequestWithContext(ctx, subj, data)
		if err != nil {
			return nil, err
		}

		// Parse reply palyload.
		rep := &enc.RPCReply{
			Result: method.NewOutput(),
		}
		if err := encoder.DecodeReply(msg.Data, rep); err != nil {
			return nil, err
		}

		// Return.
		return rep.Result, rep.Error
	}

	// Decorate handler with middlewares.
	for i := len(client.mws) - 1; i >= 0; i-- {
		handler = client.mws[i](handler)
	}
	return handler
}

// Close implements RPCClient interface.
func (client *NatsRPCClient) Close() error {
	client.mu.Lock()
	conn := client.conn
	client.conn = nil
	client.mu.Unlock()

	if conn == nil {
		return ErrClientClosed
	}
	return nil
}

func chooseServerEncoder(encoding string) enc.RPCServerEncoder {
	switch encoding {
	case "json":
		return enc.JSONServerEncoder{}
	default:
		return enc.PBServerEncoder{}
	}
}

func chooseClientEncoder(encoding string) enc.RPCClientEncoder {
	switch encoding {
	case "json":
		return enc.JSONClientEncoder{}
	default:
		return enc.PBClientEncoder{}
	}
}
