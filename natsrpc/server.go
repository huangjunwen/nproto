package natsrpc

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/taskrunner"
	"github.com/huangjunwen/golibs/taskrunner/limitedrunner"
	nats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	. "github.com/huangjunwen/nproto/v2"
	. "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
	nppb "github.com/huangjunwen/nproto/v2/pb"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

type Server struct {
	// Immutable fields.
	nc            *nats.Conn
	ctx           context.Context
	subjectPrefix string                // subject prefix in nats namespace
	group         string                // server group
	encoders      map[string]Encoder    // default encoders: encoderName -> Encoder
	runner        taskrunner.TaskRunner // runner for handlers
	logger        logr.Logger

	mu sync.Mutex
	// Mutable fields.
	closed     bool
	subs       map[string]*nats.Subscription // svcName -> *nats.Subscription
	methodMaps map[string]*methodMap         // svcName -> *methodMap
}

type ServerOption func(*Server) error

// methodMap stores method info for a service.
type methodMap struct {
	// XXX: sync.Map ?
	mu sync.RWMutex
	v  map[string]*methodInfo
}

type methodInfo struct {
	spec     *RPCSpec
	handler  RPCHandler
	encoders map[string]Encoder // supported encoders for this method
}

var (
	_ RPCServer = (*Server)(nil)
)

func NewServer(nc *nats.Conn, opts ...ServerOption) (srv *Server, err error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrNCMaxReconnect
	}

	server := &Server{
		logger:        logr.Nop,
		subjectPrefix: DefaultSubjectPrefix,
		group:         DefaultGroup,
		nc:            nc,
		ctx:           context.Background(),
		subs:          make(map[string]*nats.Subscription),
		methodMaps:    make(map[string]*methodMap),
	}

	server.runner, err = limitedrunner.New()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			// XXX: server.runner maybe not the initial runner since ServerOption
			// can change it.
			server.runner.Close()
		}
	}()

	if err = ServerOptEncoders(DefaultServerEncoders...)(server); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err = opt(server); err != nil {
			return nil, err
		}
	}

	return server, nil
}

func (server *Server) RegistHandler(spec *RPCSpec, handler RPCHandler) error {
	return server.registHandler(spec, handler, server.encoders)
}

func (server *Server) AltEncoderServer(encoders ...Encoder) (RPCServer, error) {

	if len(encoders) == 0 {
		return nil, fmt.Errorf("natsrpc.Server.AltEncoderServer no encoder?")
	}

	encoders_ := map[string]Encoder{}
	for _, encoder := range encoders {
		encoders_[encoder.EncoderName()] = encoder
	}

	return RPCServerFunc(func(spec *RPCSpec, handler RPCHandler) error {
		return server.registHandler(spec, handler, encoders_)
	}), nil

}

func (server *Server) MustAltEncoderServer(encoders ...Encoder) RPCServer {
	ret, err := server.AltEncoderServer(encoders...)
	if err != nil {
		panic(err)
	}
	return ret
}

func (server *Server) Close() error {

	server.mu.Lock()
	defer server.mu.Unlock()

	if server.closed {
		return ErrServerClosed
	}

	server.closed = true
	server.runner.Close()

	if len(server.subs) != 0 {
		for _, sub := range server.subs {
			sub.Unsubscribe()
		}
	}

	return nil
}

func (server *Server) registHandler(spec *RPCSpec, handler RPCHandler, encoders map[string]Encoder) error {

	if err := spec.Validate(); err != nil {
		return err
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	if server.closed {
		return ErrServerClosed
	}

	svcName := spec.SvcName
	sub, ok := server.subs[svcName]

	// No subscription means that it's the first method (of the service) registed.
	if !ok {
		mm := newMethodMap()
		msgHandler := server.msgHandler(svcName, mm)

		// Real subscription.
		var err error
		sub, err = server.nc.QueueSubscribe(
			subjectFormat(server.subjectPrefix, svcName, ">"),
			server.group,
			msgHandler,
		)
		if err != nil {
			return err
		}

		// Store subscription and method map.
		server.subs[svcName] = sub
		server.methodMaps[svcName] = mm
	}

	server.methodMaps[svcName].Regist(spec, handler, encoders)
	return nil

}

func (server *Server) msgHandler(svcName string, mm *methodMap) nats.MsgHandler {

	parser := subjectParser(server.subjectPrefix, svcName)

	return func(msg *nats.Msg) {

		ok, methodName := parser(msg.Subject)
		if !ok {
			// XXX: basically impossible branch
			server.logger.Error(
				nil,
				"svc got unexpected subject",
				"svc", svcName,
				"subject", msg.Subject,
			)
			return
		}

		var _logger logr.Logger
		logger := func() logr.Logger {
			if _logger == nil {
				_logger = server.logger.WithValues("svc", svcName, "method", methodName)
			}
			return _logger
		}

		reply := func(resp *nppb.NatsRPCResponse) {
			respData, err := proto.Marshal(resp)
			if err != nil {
				// XXX: basically impossible branch
				logger().Error(err, "marshal response error")
				return
			}

			err = server.nc.Publish(msg.Reply, respData)
			if err != nil {
				logger().Error(err, "publish response error")
				return
			}
			return
		}

		replyError := func(err error) {
			switch err.(type) {
			case *Error:
				e := &nppb.Error{}
				e.From(err.(*Error))
				reply(&nppb.NatsRPCResponse{
					Out: &nppb.NatsRPCResponse_Err{
						Err: e,
					},
				})

			default:
				reply(&nppb.NatsRPCResponse{
					Out: &nppb.NatsRPCResponse_PlainErr{
						PlainErr: err.Error(),
					},
				})
			}
		}

		replyNprotoError := func(code ErrorCode, msg string, args ...interface{}) {
			a := make([]interface{}, 0, len(args)+2)
			a = append(a, svcName, methodName)
			a = append(a, args...)
			replyError(Errorf(
				code,
				"natsrpc.Server(%s::%s) "+msg,
				a...,
			))
		}

		if err := server.runner.Submit(func() {

			info := mm.Lookup(methodName)
			if info == nil {
				replyNprotoError(RPCMethodNotFound, "method not found")
				return
			}

			req := &nppb.NatsRPCRequest{}
			if err := proto.Unmarshal(msg.Data, req); err != nil {
				replyNprotoError(RPCRequestDecodeError, "unmarshal request error: %s", err.Error())
				return
			}

			encoderName := req.Input.EncoderName
			encoder, ok := info.encoders[encoderName]
			if !ok {
				replyNprotoError(RPCRequestDecodeError, "method does not support encoder %s", encoderName)
				return
			}

			input := info.spec.NewInput()
			if err := encoder.DecodeData(bytes.NewReader(req.Input.Bytes), input); err != nil {
				replyNprotoError(RPCRequestDecodeError, "decode input error: %s", err.Error())
				return
			}

			// Setup context.
			ctx := server.ctx
			if req.MetaData != nil {
				ctx = npmd.NewIncomingContextWithMD(ctx, req.MetaData.To())
			}
			if req.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout))
				defer cancel()
			}

			output, err := info.handler(ctx, input)
			if err != nil {
				replyError(err)
				return
			}

			if err := info.spec.AssertOutputType(output); err != nil {
				logger().Error(err, "assert output error")
				replyNprotoError(RPCResponseEncodeError, "assert output error: %s", err.Error())
				return
			}

			w := &bytes.Buffer{}
			if err := encoder.EncodeData(w, output); err != nil {
				logger().Error(err, "encode output error")
				replyNprotoError(RPCResponseEncodeError, "encode output error: %s", err.Error())
				return
			}

			reply(&nppb.NatsRPCResponse{
				Out: &nppb.NatsRPCResponse_Output{
					Output: &nppb.RawData{
						EncoderName: encoder.EncoderName(),
						Bytes:       w.Bytes(),
					},
				},
			})

		}); err != nil {

			logger().Error(err, "submit task error")
			replyError(fmt.Errorf("natsrpc.Server submit task error: %s", err.Error()))

		}

	}

}

func newMethodMap() *methodMap {
	return &methodMap{
		v: make(map[string]*methodInfo),
	}
}

func (mm *methodMap) Lookup(methodName string) *methodInfo {
	mm.mu.RLock()
	info := mm.v[methodName]
	mm.mu.RUnlock()
	return info
}

func (mm *methodMap) Regist(spec *RPCSpec, handler RPCHandler, encoders map[string]Encoder) {

	if handler == nil {
		mm.mu.Lock()
		delete(mm.v, spec.MethodName)
		mm.mu.Unlock()
		return
	}

	info := &methodInfo{
		spec:     spec,
		handler:  handler,
		encoders: encoders,
	}
	mm.mu.Lock()
	mm.v[spec.MethodName] = info
	mm.mu.Unlock()
	return

}
