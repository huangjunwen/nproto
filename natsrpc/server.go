package natsrpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/taskrunner"
	"github.com/huangjunwen/golibs/taskrunner/limitedrunner"
	nats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	npenc "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
	nppbmd "github.com/huangjunwen/nproto/v2/pb/md"
	nppbnatsrpc "github.com/huangjunwen/nproto/v2/pb/natsrpc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

type ServerConn struct {
	// Immutable fields.
	nc            *nats.Conn
	ctx           context.Context
	subjectPrefix string                // subject prefix in nats namespace
	group         string                // server group
	runner        taskrunner.TaskRunner // runner for handlers
	logger        logr.Logger

	mu sync.Mutex // to protect mutable fields

	// Mutable fields.
	closed     bool
	subs       map[string]*nats.Subscription // svcName -> *nats.Subscription
	methodMaps map[string]*methodMap         // svcName -> *methodMap
}

type ServerConnOption func(*ServerConn) error

// methodMap stores method info for a service.
type methodMap struct {
	// XXX: sync.Map ?
	mu sync.RWMutex
	v  map[string]*methodInfo
}

type methodInfo struct {
	spec    RPCSpec
	handler RPCHandler
	decoder npenc.Decoder
	encoder npenc.Encoder
}

func NewServerConn(nc *nats.Conn, opts ...ServerConnOption) (sc *ServerConn, err error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrNCMaxReconnect
	}

	serverConn := &ServerConn{
		nc:            nc,
		ctx:           context.Background(),
		subjectPrefix: DefaultSubjectPrefix,
		group:         DefaultGroup,
		runner:        limitedrunner.Must(),
		logger:        logr.Nop,
		subs:          make(map[string]*nats.Subscription),
		methodMaps:    make(map[string]*methodMap),
	}

	defer func() {
		if err != nil {
			// XXX: server.runner maybe not the initial runner since ServerOption
			// can change it.
			serverConn.runner.Close()
		}
	}()

	for _, opt := range opts {
		if err = opt(serverConn); err != nil {
			return nil, err
		}
	}

	return serverConn, nil
}

func (sc *ServerConn) Server(decoder npenc.Decoder, encoder npenc.Encoder) RPCServerFunc {
	return func(spec RPCSpec, handler RPCHandler) error {
		return sc.registHandler(spec, handler, decoder, encoder)
	}
}

func (sc *ServerConn) Close() error {

	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.closed {
		return ErrClosed
	}

	sc.closed = true
	sc.runner.Close()

	if len(sc.subs) != 0 {
		for _, sub := range sc.subs {
			sub.Unsubscribe()
		}
	}

	return nil
}

func (sc *ServerConn) registHandler(spec RPCSpec, handler RPCHandler, decoder npenc.Decoder, encoder npenc.Encoder) error {

	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.closed {
		return ErrClosed
	}

	svcName := spec.SvcName()
	sub, ok := sc.subs[svcName]

	// No subscription means that it's the first method (of the service) registed.
	if !ok {
		mm := newMethodMap()
		msgHandler := sc.msgHandler(svcName, mm)

		// Real subscription.
		var err error
		sub, err = sc.nc.QueueSubscribe(
			subjectFormat(sc.subjectPrefix, svcName, ">"),
			sc.group,
			msgHandler,
		)
		if err != nil {
			return err
		}

		// Store subscription and method map.
		sc.subs[svcName] = sub
		sc.methodMaps[svcName] = mm
	}

	sc.methodMaps[svcName].Regist(spec, handler, decoder, encoder)
	return nil

}

func (sc *ServerConn) msgHandler(svcName string, mm *methodMap) nats.MsgHandler {

	parser := subjectParser(sc.subjectPrefix, svcName)

	return func(msg *nats.Msg) {

		ok, methodName := parser(msg.Subject)
		if !ok {
			// XXX: basically impossible branch
			panic(fmt.Errorf("svc %q got unexpected subject %q", svcName, msg.Subject))
		}

		errorf := func(code RPCErrorCode, msg string, args ...interface{}) error {
			a := make([]interface{}, 0, len(args)+2)
			a = append(a, svcName, methodName)
			a = append(a, args...)
			return RPCErrorf(
				code,
				"natsrpc::server[%s::%s] "+msg,
				a...,
			)
		}

		var logger func() logr.Logger
		{
			var l logr.Logger
			logger = func() logr.Logger {
				if l == nil {
					l = sc.logger.WithValues("svc", svcName, "method", methodName)
				}
				return l
			}
		}

		reply := func(resp *nppbnatsrpc.Response) {
			respData, err := proto.Marshal(resp)
			if err != nil {
				// XXX: basically impossible branch
				logger().Error(err, "marshal response error")
				return
			}

			err = sc.nc.Publish(msg.Reply, respData)
			if err != nil {
				logger().Error(err, "publish response error")
				return
			}
			return
		}

		replyError := func(err error) {
			switch e := err.(type) {
			case *RPCError:
				reply(&nppbnatsrpc.Response{
					Type:       nppbnatsrpc.Response_RpcErr,
					ErrCode:    int32(e.Code),
					ErrMessage: e.Message,
				})

			default:
				reply(&nppbnatsrpc.Response{
					Type:       nppbnatsrpc.Response_Err,
					ErrMessage: err.Error(),
				})
			}
		}

		if err := sc.runner.Submit(func() {
			defer func() {
				if e := recover(); e != nil {
					err, ok := e.(error)
					if !ok {
						err = fmt.Errorf("%+v", e)
					}
					logger().Error(err, "handler panic")
				}
			}()

			info := mm.Lookup(methodName)
			if info == nil {
				replyError(errorf(MethodNotFound, "method not found"))
				return
			}

			req := &nppbnatsrpc.Request{}
			if err := proto.Unmarshal(msg.Data, req); err != nil {
				replyError(errorf(DecodeRequestError, "unmarshal reqeust error %s", err.Error()))
				return
			}

			input := info.spec.NewInput()
			if err := info.decoder.DecodeData(req.InputFormat, req.InputBytes, input); err != nil {
				replyError(errorf(DecodeRequestError, "decode input error %s", err.Error()))
				return
			}

			// Setup context.
			ctx := sc.ctx
			if len(req.MetaData) != 0 {
				ctx = npmd.NewIncomingContextWithMD(ctx, nppbmd.MetaData(req.MetaData))
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

			if err := AssertOutputType(info.spec, output); err != nil {
				logger().Error(err, "assert output error")
				replyError(errorf(EncodeResponseError, "assert output error %s", err.Error()))
				return
			}

			resp := &nppbnatsrpc.Response{
				Type:         nppbnatsrpc.Response_Output,
				OutputFormat: req.InputFormat,
			}
			if err := info.encoder.EncodeData(output, &resp.OutputFormat, &resp.OutputBytes); err != nil {
				logger().Error(err, "encode output error")
				replyError(errorf(EncodeResponseError, "encode output error %s", err.Error()))
				return
			}

			reply(resp)

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

func (mm *methodMap) Regist(spec RPCSpec, handler RPCHandler, decoder npenc.Decoder, encoder npenc.Encoder) {

	if handler == nil {
		mm.mu.Lock()
		delete(mm.v, spec.MethodName())
		mm.mu.Unlock()
		return
	}

	info := &methodInfo{
		spec:    spec,
		handler: handler,
		decoder: decoder,
		encoder: encoder,
	}
	mm.mu.Lock()
	mm.v[spec.MethodName()] = info
	mm.mu.Unlock()
	return

}
