package natsrpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	npenc "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
	nppbmd "github.com/huangjunwen/nproto/v2/pb/md"
	nppbnatsrpc "github.com/huangjunwen/nproto/v2/pb/natsrpc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

// ClientConn wraps nats.Conn into 'client side rpc connection'.
type ClientConn struct {
	// Immutable fields.
	subjectPrefix string
	timeout       time.Duration
	nc            *nats.Conn
}

// ClientConnOption is option in creating ClientConn.
type ClientConnOption func(*ClientConn) error

// NewClientConn creates a new ClientConn. `nc` must have MaxReconnect < 0
// (e.g. never give up trying to reconnect).
func NewClientConn(nc *nats.Conn, opts ...ClientConnOption) (*ClientConn, error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrNCMaxReconnect
	}

	cc := &ClientConn{
		subjectPrefix: DefaultSubjectPrefix,
		timeout:       DefaultClientTimeout,
		nc:            nc,
	}
	for _, opt := range opts {
		if err := opt(cc); err != nil {
			return nil, err
		}
	}

	return cc, nil
}

// Client creates an rpc client using specified encoder and decoder.
func (cc *ClientConn) Client(encoder npenc.Encoder, decoder npenc.Decoder) RPCClientFunc {
	return func(spec RPCSpec) RPCHandler {
		return cc.makeHandler(spec, encoder, decoder)
	}
}

func (cc *ClientConn) makeHandler(spec RPCSpec, encoder npenc.Encoder, decoder npenc.Decoder) RPCHandler {

	errorf := func(code RPCErrorCode, msg string, args ...interface{}) error {
		a := make([]interface{}, 0, len(args)+2)
		a = append(a, spec.SvcName(), spec.MethodName())
		a = append(a, args...)
		return RPCErrorf(
			code,
			"natsrpc::client[%s::%s] "+msg,
			a...,
		)
	}

	return func(ctx context.Context, input interface{}) (interface{}, error) {

		if err := AssertInputType(spec, input); err != nil {
			return nil, errorf(EncodeRequestError, "assert input type error %s", err.Error())
		}

		req := &nppbnatsrpc.Request{
			MetaData: nppbmd.NewMetaData(npmd.MDFromOutgoingContext(ctx)),
		}

		if err := encoder.EncodeData(input, &req.InputFormat, &req.InputBytes); err != nil {
			return nil, errorf(EncodeRequestError, "encode input error %s", err.Error())
		}

		dl, ok := ctx.Deadline()
		if !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, cc.timeout)
			defer cancel()
			dl, ok = ctx.Deadline()
			if !ok {
				panic("No ctx.Deadline()")
			}
		}
		dur := dl.Sub(time.Now())
		if dur <= 0 {
			return nil, context.DeadlineExceeded
		}
		req.Timeout = uint64(dur)

		reqData, err := proto.Marshal(req)
		if err != nil {
			return nil, errorf(EncodeRequestError, "marshal request error %s", err.Error())
		}

		respMsg, err := cc.nc.RequestWithContext(
			ctx,
			subjectFormat(cc.subjectPrefix, spec.SvcName(), spec.MethodName()),
			reqData,
		)
		if err != nil {
			return nil, err
		}

		resp := &nppbnatsrpc.Response{}
		if err := proto.Unmarshal(respMsg.Data, resp); err != nil {
			return nil, errorf(DecodeResponseError, "unmarshal response error %s", err.Error())
		}

		switch resp.Type {
		case nppbnatsrpc.Response_Output:
			if resp.OutputFormat != req.InputFormat {
				return nil, errorf(
					DecodeResponseError,
					"expect output format %s, but got %s",
					req.InputFormat,
					resp.OutputFormat,
				)
			}

			output := spec.NewOutput()
			if err := decoder.DecodeData(resp.OutputFormat, resp.OutputBytes, output); err != nil {
				return nil, errorf(
					DecodeResponseError,
					"decode output error: %s",
					err.Error(),
				)
			}

			return output, nil

		case nppbnatsrpc.Response_RpcErr:
			return nil, &RPCError{
				Code:    RPCErrorCode(resp.ErrCode),
				Message: resp.ErrMessage,
			}

		case nppbnatsrpc.Response_Err:
			return nil, errors.New(resp.ErrMessage)

		default:
			panic(fmt.Errorf("Unexpected branch"))
		}

	}
}
