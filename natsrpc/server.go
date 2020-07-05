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
	// Option fields.
	logger        logr.Logger
	runner        taskrunner.TaskRunner // runner for handlers
	subjectPrefix string                // subject prefix in nats namespace
	group         string                // server group
	encoders      map[string]Encoder    // supported encoders: encoderName -> Encoder
	encoderNames  map[string]struct{}

	// Immutable fields.
	nc  *nats.Conn
	ctx context.Context

	// Mutable fields.
	mu         sync.Mutex
	closed     bool
	subs       map[string]*nats.Subscription // svcName -> *nats.Subscription
	methodMaps map[string]*methodMap         // svcName -> *methodMap
}

type ServerOption func(*Server) error

var (
	_ RPCServer = (*Server)(nil)
)

func NewServer(nc *nats.Conn, opts ...ServerOption) (server *Server, err error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrNCMaxReconnect
	}

	server = &Server{
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

	if err := ServerOptEncoders(DefaultServerEncoders...)(server); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt(server); err != nil {
			return nil, err
		}
	}

	return server, nil
}

func (server *Server) RegistHandler(spec *RPCSpec, handler RPCHandler) error {
	return server.registHandler(spec, handler, server.encoderNames)
}

func (server *Server) SubServer(encoderNames ...string) (RPCServer, error) {

	encoderNameSet := map[string]struct{}{}
	for _, encoderName := range encoderNames {
		if _, ok := server.encoders[encoderName]; !ok {
			return nil, fmt.Errorf("Encoder %s not found", encoderName)
		}
		encoderNameSet[encoderName] = struct{}{}
	}

	return RPCServerFunc(func(spec *RPCSpec, handler RPCHandler) error {
		return server.registHandler(spec, handler, encoderNameSet)
	}), nil

}

func (server *Server) MustSubServer(encoderNames ...string) RPCServer {
	ret, err := server.SubServer(encoderNames...)
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

func (server *Server) registHandler(spec *RPCSpec, handler RPCHandler, encoderNames map[string]struct{}) error {

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

	server.methodMaps[svcName].RegistHandler(spec, handler, encoderNames)
	return nil

}

func (server *Server) msgHandler(svcName string, mm *methodMap) nats.MsgHandler {

	parser := subjectParser(server.subjectPrefix, svcName)

	return func(msg *nats.Msg) {
		subj := msg.Reply

		if err := server.runner.Submit(func() {

			ok, methodName := parser(msg.Subject)
			if !ok {
				server.logger.Error(nil, "natsrpc.Server unexpected rpc subject", "subject", msg.Subject)
				return
			}

			req := &nppb.NatsRPCRequest{}
			if err := proto.Unmarshal(msg.Data, req); err != nil {
				server.replyError(subj, Errorf(
					ProtocolError,
					"natsrpc.Server %s::%s unmarshal request error: %s",
					svcName,
					methodName,
					err.Error(),
				))
				return
			}

			info := mm.Get()[methodName]
			if info == nil {
				server.replyError(subj, Errorf(
					PayloadError,
					"natsrpc.Server %s::%s not found",
					svcName,
					methodName,
				))
				return
			}

			encoderName := req.Input.EncoderName
			if _, ok := info.EncoderNames[encoderName]; !ok {
				server.replyError(subj, Errorf(
					PayloadError,
					"natsrpc.Server %s::%s does not support encoder %s",
					svcName,
					methodName,
					encoderName,
				))
				return
			}

			encoder := server.encoders[encoderName]
			input := info.Spec.NewInput()
			if err := encoder.DecodeData(bytes.NewReader(req.Input.Bytes), input); err != nil {
				server.replyError(subj, Errorf(
					PayloadError,
					"natsrpc.Server %s::%s decode input error: %s",
					svcName,
					methodName,
					err.Error(),
				))
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

			output, err := info.Handler(ctx, input)
			if err != nil {
				server.replyError(subj, err)
			} else {
				server.replyOutput(subj, encoder, output)
			}

		}); err != nil {

			server.replyError(subj, fmt.Errorf("natsrpc.Server submit task error: %s", err.Error()))
			server.logger.Error(err, "natsrpc.Server submit task error")

		}

	}

}

func (server *Server) replyOutput(subj string, encoder Encoder, output interface{}) {

	w := &bytes.Buffer{}
	if err := encoder.EncodeData(w, output); err != nil {
		server.replyError(subj, Errorf(PayloadError, "natsrpc.Server encode output error: %s", err.Error()))
		server.logger.Error(err, "natsrpc.Server encode output error")
		return
	}

	server.reply(subj, &nppb.NatsRPCResponse{
		Out: &nppb.NatsRPCResponse_Output{
			Output: &nppb.RawData{
				EncoderName: encoder.Name(),
				Bytes:       w.Bytes(),
			},
		},
	})

}

func (server *Server) replyError(subj string, err error) {

	switch err.(type) {
	case *Error:
		e := &nppb.Error{}
		e.From(err.(*Error))
		server.reply(subj, &nppb.NatsRPCResponse{
			Out: &nppb.NatsRPCResponse_Err{
				Err: e,
			},
		})

	default:
		server.reply(subj, &nppb.NatsRPCResponse{
			Out: &nppb.NatsRPCResponse_PlainErr{
				PlainErr: err.Error(),
			},
		})
	}

}

func (server *Server) reply(subj string, resp *nppb.NatsRPCResponse) {

	respData, err := proto.Marshal(resp)
	if err != nil {
		server.logger.Error(err, "natsrpc.Server marshal response error")
		return
	}

	err = server.nc.Publish(subj, respData)
	if err != nil {
		server.logger.Error(err, "natsrpc.Server publish response error")
		return
	}
	return

}
