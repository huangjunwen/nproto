package nprpc

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

	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/nprpc/enc"
)

var (
	DefaultSubjectPrefix = "nprpc"
	DefaultGroup         = "def"
	DefaultEncoding      = "pb"
)

var (
	ErrMaxReconnect = errors.New("nproto.nprpc: nc should have MaxReconnects < 0")
	ErrServerClosed = errors.New("nproto.nprpc.NatsRPCServer: Server closed")
	ErrDupSvcName   = func(svcName string) error {
		return fmt.Errorf("nproto.nprpc.NatsRPCServer: Duplicated service %+q", svcName)
	}
	ErrDupMethodName = func(methodName string) error {
		return fmt.Errorf("nproto.nprpc.NatsRPCServer: Duplicated method %+q", methodName)
	}
	ErrClientClosed = errors.New("nproto.nprpc.NatsRPCClient: Client closed")
)

// NatsRPCServer implements RPCServer.
type NatsRPCServer struct {
	// Options.
	logger        zerolog.Logger
	subjectPrefix string
	group         string // server group

	// Mutable fields.
	mu   sync.RWMutex
	nc   *nats.Conn                    // nil if closed
	svcs map[string]*nats.Subscription // svcName -> Subscription

	wg sync.WaitGroup // wait group for active handlers.
}

// NatsRPCClient implements RPCClient.
type NatsRPCClient struct {
	// Options.
	subjectPrefix string // subject prefix
	encoding      string // rpc encoding

	// Mutable fields.
	mu sync.RWMutex
	nc *nats.Conn // nil if closed
}

// ServerOption is option in creating NatsRPCServer.
type ServerOption func(*NatsRPCServer) error

// ClientOption is option in creating NatsRPCClient.
type ClientOption func(*NatsRPCClient) error

var (
	_ nproto.RPCServer = (*NatsRPCServer)(nil)
	_ nproto.RPCClient = (*NatsRPCClient)(nil)
)

// NewNatsRPCServer creates a new NatsRPCServer. `nc` must have MaxReconnects < 0 set (e.g. Always reconnect).
func NewNatsRPCServer(nc *nats.Conn, opts ...ServerOption) (*NatsRPCServer, error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrMaxReconnect
	}

	server := &NatsRPCServer{
		subjectPrefix: DefaultSubjectPrefix,
		group:         DefaultGroup,
		logger:        zerolog.Nop(),
		nc:            nc,
		svcs:          make(map[string]*nats.Subscription),
	}
	for _, opt := range opts {
		if err := opt(server); err != nil {
			return nil, err
		}
	}

	return server, nil
}

// RegistSvc implements RPCServer interface.
func (server *NatsRPCServer) RegistSvc(svcName string, methods map[*nproto.RPCMethod]nproto.RPCHandler) (err error) {
	// Create msg handler.
	handler, err := server.msgHandler(svcName, methods)
	if err != nil {
		return
	}

	// Set subscription placeholder.
	if err = func() error {
		server.mu.Lock()
		defer server.mu.Unlock()

		if server.nc == nil {
			return ErrServerClosed
		}
		if _, ok := server.svcs[svcName]; ok {
			return ErrDupSvcName(svcName)
		}
		server.svcs[svcName] = nil // NOTE: Set a placeholder so that other can't regist the same name.
		return nil
	}(); err != nil {
		return
	}
	defer func() {
		// Release the placeholder if any error.
		if err != nil {
			server.mu.Lock()
			delete(server.svcs, svcName)
			server.mu.Unlock()
		}
	}()

	// Real subscription.
	sub, err := server.nc.QueueSubscribe(
		fmt.Sprintf("%s.%s.>", server.subjectPrefix, svcName),
		server.group,
		handler,
	)
	if err != nil {
		return
	}
	defer func() {
		// Unsubscribe if any error.
		if err != nil {
			sub.Unsubscribe()
		}
	}()

	// Set subscription.
	server.mu.Lock()
	if server.nc == nil {
		err = ErrServerClosed
	} else {
		server.svcs[svcName] = sub
	}
	server.mu.Unlock()
	return
}

// DeregistSvc implements RPCServer interface.
func (server *NatsRPCServer) DeregistSvc(svcName string) error {
	// Pop svcName.
	server.mu.Lock()
	sub := server.svcs[svcName]
	delete(server.svcs, svcName)
	server.mu.Unlock()

	// Unsubscribe.
	if sub != nil {
		return sub.Unsubscribe()
	}
	return nil
}

// Close stops the server and deregist all registed services, then wait all active handlers to finish.
func (server *NatsRPCServer) Close() error {
	// Set nc to nil to indicate close.
	server.mu.Lock()
	nc := server.nc
	svcs := server.svcs
	server.nc = nil
	server.svcs = nil
	server.mu.Unlock()

	// Already closed.
	if nc == nil {
		return ErrServerClosed
	}

	// Unsubscribe.
	if len(svcs) != 0 {
		for _, sub := range svcs {
			sub.Unsubscribe()
		}
	}

	// Wait all active handlers to finish.
	server.wg.Wait()
	return nil
}

func (server *NatsRPCServer) msgHandler(svcName string, methods map[*nproto.RPCMethod]nproto.RPCHandler) (nats.MsgHandler, error) {
	// Method name -> method
	methodNames := make(map[string]*nproto.RPCMethod)
	for method, _ := range methods {
		if _, found := methodNames[method.Name]; found {
			return nil, ErrDupMethodName(method.Name)
		}
		methodNames[method.Name] = method
	}

	// Subject prefix.
	prefix := fmt.Sprintf("%s.%s.", server.subjectPrefix, svcName)

	return func(msg *nats.Msg) {
		// If closed, ignore this msg.
		server.mu.RLock()
		nc := server.nc
		server.mu.RUnlock()
		if nc == nil {
			return
		}

		go func() {
			server.wg.Add(1)
			defer server.wg.Done()

			// Subject should be in the form of "subjectPrefix.svcName.enc.method".
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
				server.replyError(msg.Reply, fmt.Errorf("Method %+q not found", methodName), encoding)
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
			ctx := nproto.NewRPCCtx(context.Background(), svcName, method, req.Passthru)
			if req.Timeout != nil {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, *req.Timeout)
				defer cancel()
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

	var (
		data []byte
		err  error
	)

	// Check closed.
	server.mu.RLock()
	nc := server.nc
	server.mu.RUnlock()
	if nc == nil {
		err = ErrServerClosed
		goto Err
	}

	// Encode reply.
	data, err = chooseServerEncoder(encoding).EncodeReply(r)
	if err != nil {
		goto Err
	}

	// Publish reply.
	err = nc.Publish(subj, data)
	if err != nil {
		goto Err
	}
	return

Err:
	server.logger.Error().Err(err).Msg("")
	return
}

// NewNatsRPCClient creates a new NatsRPCClient. `nc` must have MaxReconnects < 0 set (e.g. Always reconnect).
func NewNatsRPCClient(nc *nats.Conn, opts ...ClientOption) (*NatsRPCClient, error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrMaxReconnect
	}

	client := &NatsRPCClient{
		subjectPrefix: DefaultSubjectPrefix,
		encoding:      DefaultEncoding,
		nc:            nc,
	}
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// MakeHandler implements RPCClient interface.
func (client *NatsRPCClient) MakeHandler(svcName string, method *nproto.RPCMethod) nproto.RPCHandler {

	encoder := chooseClientEncoder(client.encoding)
	subj := fmt.Sprintf("%s.%s.%s.%s", client.subjectPrefix, svcName, client.encoding, method.Name)

	return func(ctx context.Context, input proto.Message) (proto.Message, error) {

		// Get conn and check closed.
		client.mu.RLock()
		nc := client.nc
		client.mu.RUnlock()
		if nc == nil {
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
		passthru := nproto.Passthru(ctx)
		if len(passthru) > 0 {
			req.Passthru = passthru
		}

		// Encode request.
		data, err := encoder.EncodeRequest(req)
		if err != nil {
			return nil, err
		}

		// Send request.
		msg, err := nc.RequestWithContext(ctx, subj, data)
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
}

// Close closes the client.
func (client *NatsRPCClient) Close() error {
	client.mu.Lock()
	nc := client.nc
	client.nc = nil
	client.mu.Unlock()

	if nc == nil {
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
