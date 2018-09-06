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

	"github.com/huangjunwen/nproto/rpc/enc"
)

var (
	DefaultSubjectPrefix = "nprpc"
	DefaultGroup         = "def"
	DefaultEncoding      = "pb"
)

var (
	ErrMaxReconnect = errors.New("nproto.nprpc.natsrpc: nc should have MaxReconnects < 0")
	ErrServerClosed = errors.New("nproto.nprpc.NatsRPCServer: Server closed")
	ErrClientClosed = errors.New("nproto.nprpc.NatsRPCClient: Client closed")
)

// NatsRPCServer implements RPCServer.
type NatsRPCServer struct {
	// Options.
	subjPrefix string         // subject prefix
	group      string         // server group
	logger     zerolog.Logger // logger

	// Mutable fields.
	mu   sync.RWMutex
	nc   *nats.Conn                    // nil if closed
	subs map[string]*nats.Subscription // svcName -> Subscription
}

// NatsRPCClient implements RPCClient.
type NatsRPCClient struct {
	// Options.
	subjPrefix string // subject prefix
	encoding   string // rpc encoding

	// Mutable fields.
	mu sync.RWMutex
	nc *nats.Conn // nil if closed
}

// NatsRPCServerOption is option in creating NatsRPCServer.
type NatsRPCServerOption func(*NatsRPCServer) error

// NatsRPCClientOption is option in creating NatsRPCClient.
type NatsRPCClientOption func(*NatsRPCClient) error

var (
	_ RPCServer = (*NatsRPCServer)(nil)
	_ RPCClient = (*NatsRPCClient)(nil)
)

// NewNatsRPCServer creates a new NatsRPCServer. `nc` should have MaxReconnects < 0 set (e.g. Always reconnect).
func NewNatsRPCServer(nc *nats.Conn, opts ...NatsRPCServerOption) (*NatsRPCServer, error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrMaxReconnect
	}

	server := &NatsRPCServer{
		subjPrefix: DefaultSubjectPrefix,
		group:      DefaultGroup,
		logger:     zerolog.Nop(),
		nc:         nc,
		subs:       make(map[string]*nats.Subscription),
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
	// Create msg handler.
	h, err := server.msgHandler(svcName, methods)
	if err != nil {
		return err
	}

	// First (read) lock: get nc and do some checks.
	server.mu.RLock()
	if server.nc == nil {
		server.mu.RUnlock()
		return ErrServerClosed
	}
	if server.subs[svcName] != nil {
		server.mu.RUnlock()
		return fmt.Errorf("nproto.nprpc.NatsRPCServer: Duplicated service name %+q", svcName)
	}
	nc := server.nc
	server.mu.RUnlock()

	// Subscribe.
	subs, err := natsQueueSubscribe(
		nc,
		fmt.Sprintf("%s.%s.>", server.subjPrefix, svcName),
		server.group,
		h,
	)
	if err != nil {
		return err
	}

	// Second (write) lock: set subscription
	server.mu.Lock()
	if server.nc == nil {
		server.mu.Unlock()
		subs.Unsubscribe()
		return ErrServerClosed
	}
	if server.subs[svcName] != nil {
		server.mu.Unlock()
		subs.Unsubscribe()
		return fmt.Errorf("nproto.nprpc.NatsRPCServer: Duplicated service name %+q", svcName)
	}
	server.subs[svcName] = subs
	server.mu.Unlock()
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
	nc := server.nc
	subs := server.subs
	server.nc = nil
	server.subs = nil
	server.mu.Unlock()

	// Release subscriptions.
	if len(subs) != 0 {
		for _, sub := range subs {
			sub.Unsubscribe()
		}
	}

	// Multiple calls to Close.
	if nc == nil {
		return ErrServerClosed
	}
	return nil
}

func (server *NatsRPCServer) msgHandler(svcName string, methods map[*RPCMethod]RPCHandler) (nats.MsgHandler, error) {
	// Method name -> method
	methodNames := make(map[string]*RPCMethod)
	for method, _ := range methods {
		if _, found := methodNames[method.Name]; found {
			return nil, fmt.Errorf("nproto.nprpc.NatsRPCServer: Duplicated method name %+q", method.Name)
		}
		methodNames[method.Name] = method
	}

	// Subject prefix.
	prefix := server.subjPrefix + "." + svcName + "."

	return func(msg *nats.Msg) {
		cfh.Go("NatsRPCServer.msgHandler", func() {
			// Subject should be in the form of "subjPrefix.svcName.enc.method".
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

		})
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
	err = natsPublish(nc, subj, data)
	if err != nil {
		goto Err
	}
	return

Err:
	server.logger.Error().Err(err).Msg("")
	return
}

// NewNatsRPCClient creates a new NatsRPCClient. `nc` should have MaxReconnects < 0 set (e.g. Always reconnect).
func NewNatsRPCClient(nc *nats.Conn, opts ...NatsRPCClientOption) (*NatsRPCClient, error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrMaxReconnect
	}

	client := &NatsRPCClient{
		subjPrefix: DefaultSubjectPrefix,
		encoding:   DefaultEncoding,
		nc:         nc,
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
	subj := strings.Join([]string{client.subjPrefix, svcName, client.encoding, method.Name}, ".")
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
		msg, err := natsRequestWithContext(nc, ctx, subj, data)
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

// Close implements RPCClient interface.
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
		server.logger = logger.With().Str("comp", "nproto.nprpc.NatsRPCServer").Logger()
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
