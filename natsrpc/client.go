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
	nppbenc "github.com/huangjunwen/nproto/v2/pb/enc"
	nppbmd "github.com/huangjunwen/nproto/v2/pb/md"
	nppbnatsrpc "github.com/huangjunwen/nproto/v2/pb/natsrpc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

type ClientConn struct {
	// Immutable fields.
	subjectPrefix string
	timeout       time.Duration
	nc            *nats.Conn
}

type ClientConnOption func(*ClientConn) error

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

func (cc *ClientConn) Client(encoder npenc.Encoder) RPCClientFunc {
	return func(spec RPCSpec) RPCHandler {
		return cc.makeHandler(spec, encoder)
	}
}

func (cc *ClientConn) makeHandler(spec RPCSpec, encoder npenc.Encoder) RPCHandler {

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

		w := []byte{}
		if err := encoder.EncodeData(input, &w); err != nil {
			return nil, errorf(EncodeRequestError, "encode input error %s", err.Error())
		}

		req := &nppbnatsrpc.Request{
			MetaData: &nppbmd.MD{},
			Input: &nppbenc.RawData{
				EncoderName: encoder.EncoderName(),
				Bytes:       w,
			},
		}

		md := npmd.MDFromOutgoingContext(ctx)
		if md != nil {
			req.MetaData.From(md)
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

		switch out := resp.Out.(type) {
		case *nppbnatsrpc.Response_Output:
			if out.Output.EncoderName != encoder.EncoderName() {
				return nil, errorf(
					DecodeResponseError,
					"expect output encoded by %s, but got %s",
					encoder.EncoderName(),
					out.Output.EncoderName,
				)
			}

			output := spec.NewOutput()
			if err := encoder.DecodeData(out.Output.Bytes, output); err != nil {
				return nil, errorf(
					DecodeResponseError,
					"decode output error: %s",
					err.Error(),
				)
			}

			return output, nil

		case *nppbnatsrpc.Response_RpcErr:
			return nil, out.RpcErr.To()

		case *nppbnatsrpc.Response_Err:
			return nil, errors.New(out.Err)

		default:
			panic(fmt.Errorf("Unexpected branch"))
		}

	}
}
