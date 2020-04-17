package natsrpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/huangjunwen/nproto/helpers/taskrunner"
	"github.com/huangjunwen/nproto/nproto"
	enc "github.com/huangjunwen/nproto/nproto/rpcenc"
	"github.com/huangjunwen/nproto/nproto/zlog"
)

var (
	// DefaultSubjectPrefix is the default value of ServerOptSubjectPrefix/ClientOptSubjectPrefix.
	DefaultSubjectPrefix = "nprpc"
	// DefaultGroup is the default value of ServerOptGroup.
	DefaultGroup = "def"
	// DefaultEncoding is the default encoding for client.
	DefaultEncoding = "pb"
)

var (
	// ErrSvcUnavailable is returned if the service is not available to handle a request.
	ErrSvcUnavailable = errors.New("SVC_UNAVAILABLE")
	// ErrMethodNotFound is returned if the method is not found.
	ErrMethodNotFound = errors.New("METHOD_NOT_FOUND")
	// ErrInvalidRequest is returned if the request object is invalid for the method.
	ErrInvalidRequest = errors.New("INVALID_REQUEST")
)

var (
	// ErrNCMaxReconnect is returned if nc has MaxReconnects >= 0.
	ErrNCMaxReconnect = errors.New("nproto.natsrpc: nats.Conn should have MaxReconnects < 0")
	// ErrServerClosed is returned if the server has been closed.
	ErrServerClosed = errors.New("nproto.natsrpc.NatsRPCServer: Server closed")
	// ErrDupSvcName is returned if service name is duplicated.
	ErrDupSvcName = func(svcName string) error {
		return fmt.Errorf("nproto..NatsRPCServer: Duplicated service %+q", svcName)
	}
	// ErrDupMethodName is returned if method name is duplicated.
	ErrDupMethodName = func(methodName string) error {
		return fmt.Errorf("nproto.natsrpc.NatsRPCServer: Duplicated method %+q", methodName)
	}
)

// NatsRPCServer implements nproto.RPCServer interface.
type NatsRPCServer struct {
	// Options.
	logger        zerolog.Logger
	subjectPrefix string                // subject prefix in nats namespace
	group         string                // server group
	runner        taskrunner.TaskRunner // runner for handlers

	// Immutable fields.
	nc  *nats.Conn
	ctx context.Context

	// Mutable fields.
	mu     sync.Mutex
	closed bool
	svcs   map[string]*nats.Subscription // svcName -> Subscription
}

// NatsRPCClient implements nproto.RPCClient interface.
type NatsRPCClient struct {
	// Options.
	subjectPrefix string // subject prefix in nats namespace
	encoding      string // rpc encoding

	// Immutable fields.
	nc *nats.Conn
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
		return nil, ErrNCMaxReconnect
	}

	server := &NatsRPCServer{
		logger:        zerolog.Nop(),
		subjectPrefix: DefaultSubjectPrefix,
		group:         DefaultGroup,
		runner:        taskrunner.NewDefaultLimitedRunner(),
		nc:            nc,
		ctx:           context.Background(),
		svcs:          make(map[string]*nats.Subscription),
	}
	ServerOptLogger(&zlog.DefaultZLogger)(server)

	for _, opt := range opts {
		if err := opt(server); err != nil {
			return nil, err
		}
	}

	return server, nil
}

// RegistSvc implements nproto.RPCServer interface.
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

		if server.closed {
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
	if server.closed {
		err = ErrServerClosed
	} else {
		server.svcs[svcName] = sub
	}
	server.mu.Unlock()
	return
}

// DeregistSvc implements nproto.RPCServer interface.
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

// Gracefull close the server.
func (server *NatsRPCServer) Close() error {
	// Set nc to nil to indicate close.
	server.mu.Lock()
	closed := server.closed
	svcs := server.svcs
	server.closed = true
	server.svcs = nil
	server.mu.Unlock()

	// Already closed.
	if closed {
		return ErrServerClosed
	}

	// Stop task runner.
	server.runner.Close()

	// Unsubscribe.
	if len(svcs) != 0 {
		for _, sub := range svcs {
			sub.Unsubscribe()
		}
	}

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

	// Full subject is in the form of "subjectPrefix.svcName.enc.method"
	prefix := fmt.Sprintf("%s.%s.", server.subjectPrefix, svcName)
	pbPrefix := prefix + "pb."
	jsonPrefix := prefix + "json."

	return func(msg *nats.Msg) {
		// Get methodName and encoder.
		var (
			encoder    enc.RPCServerEncoder
			methodName string
		)

		for {
			methodName = strings.TrimPrefix(msg.Subject, pbPrefix)
			if len(methodName) != len(msg.Subject) {
				encoder = enc.PBServerEncoder{}
				break
			}

			methodName = strings.TrimPrefix(msg.Subject, jsonPrefix)
			if len(methodName) != len(msg.Subject) {
				encoder = enc.JSONServerEncoder{}
				break
			}

			server.logger.Error().Msgf("Unexpected RPC subject: %+q", msg.Subject)
			return
		}

		if err := server.runner.Submit(func() {
			// Check method.
			method, found := methodNames[methodName]
			if !found {
				server.replyError(msg.Reply, ErrMethodNotFound, encoder)
				return
			}
			handler := methods[method]

			// Parse request payload.
			req := &enc.RPCRequest{
				Param: method.NewInput(),
			}
			if err := encoder.DecodeRequest(msg.Data, req); err != nil {
				server.replyError(msg.Reply, ErrInvalidRequest, encoder)
				return
			}

			// Setup context.
			ctx := server.ctx
			if req.MD != nil {
				ctx = nproto.NewIncomingContextWithMD(ctx, req.MD)
			}
			if req.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, req.Timeout)
				defer cancel()
			}

			// Handle.
			result, err := handler(ctx, req.Param)
			if err != nil {
				server.replyError(msg.Reply, err, encoder)
			} else {
				server.replyResult(msg.Reply, result, encoder)
			}

		}); err != nil {
			server.replyError(msg.Reply, ErrSvcUnavailable, encoder)
			server.logger.Error().Err(err).Msg("Submit handler failed")
		}
	}, nil
}

func (server *NatsRPCServer) replyResult(subj string, result proto.Message, encoder enc.RPCServerEncoder) {
	server.reply(subj, &enc.RPCReply{
		Result: result,
	}, encoder)
}

func (server *NatsRPCServer) replyError(subj string, err error, encoder enc.RPCServerEncoder) {
	server.reply(subj, &enc.RPCReply{
		Error: err,
	}, encoder)
}

// NOTE: reply can be called after Close is called.
func (server *NatsRPCServer) reply(subj string, r *enc.RPCReply, encoder enc.RPCServerEncoder) {

	var (
		data []byte
		err  error
	)

	// Encode reply.
	data, err = encoder.EncodeReply(r)
	if err != nil {
		data, err = encoder.EncodeReply(&enc.RPCReply{
			Error: err,
		})
		if err != nil {
			goto Err
		}
	}

	// Publish reply.
	err = server.nc.Publish(subj, data)
	if err != nil {
		goto Err
	}
	return

Err:
	server.logger.Error().Err(err).Msg("Reply error")
	return
}

// NewNatsRPCClient creates a new NatsRPCClient. `nc` must have MaxReconnects < 0 set (e.g. Always reconnect).
func NewNatsRPCClient(nc *nats.Conn, opts ...ClientOption) (*NatsRPCClient, error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrNCMaxReconnect
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

// MakeHandler implements nproto.RPCClient interface.
func (client *NatsRPCClient) MakeHandler(svcName string, method *nproto.RPCMethod) nproto.RPCHandler {

	var encoder enc.RPCClientEncoder
	switch client.encoding {
	case "json":
		encoder = enc.JSONClientEncoder{}
	default:
		encoder = enc.PBClientEncoder{}
	}
	fullSubject := fmt.Sprintf("%s.%s.%s.%s", client.subjectPrefix, svcName, client.encoding, method.Name)

	return func(ctx context.Context, input proto.Message) (proto.Message, error) {

		// Construct request.
		req := &enc.RPCRequest{
			Param: input,
		}
		md := nproto.MDFromOutgoingContext(ctx)
		if md != nil {
			req.MD = md
		}
		if dl, ok := ctx.Deadline(); ok {
			dur := dl.Sub(time.Now())
			if dur <= 0 {
				return nil, context.DeadlineExceeded
			}
			req.Timeout = dur
		}

		// Encode request.
		data, err := encoder.EncodeRequest(req)
		if err != nil {
			return nil, err
		}

		// Send request.
		msg, err := client.nc.RequestWithContext(ctx, fullSubject, data)
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
