package natsrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	. "github.com/huangjunwen/nproto/v2"
	. "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
	nppb "github.com/huangjunwen/nproto/v2/pb"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

type Client struct {
	// Option fields.
	subjectPrefix string
	encoder       Encoder
	timeout       time.Duration

	// Immutable fields.
	nc *nats.Conn
}

type ClientOption func(*Client) error

var (
	_ RPCClient = (*Client)(nil)
)

func NewClient(nc *nats.Conn, opts ...ClientOption) (*Client, error) {

	if nc.Opts.MaxReconnect >= 0 {
		return nil, ErrNCMaxReconnect
	}

	client := &Client{
		subjectPrefix: DefaultSubjectPrefix,
		encoder:       DefaultClientEncoder,
		timeout:       DefaultClientTimeout,
		nc:            nc,
	}
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (client *Client) MakeHandler(spec *RPCSpec) RPCHandler {

	nprotoErrorf := func(code ErrorCode, msg string, args ...interface{}) error {
		a := make([]interface{}, 0, len(args)+2)
		a = append(a, spec.SvcName, spec.MethodName)
		a = append(a, args...)
		return Errorf(
			code,
			"natsrpc.Client(%s::%s) "+msg,
			a...,
		)
	}

	return func(ctx context.Context, input interface{}) (interface{}, error) {

		if err := spec.Validate(); err != nil {
			return nil, nprotoErrorf(NotRetryableError, "spec validate error: %s", err.Error())
		}

		if err := spec.AssertInputType(input); err != nil {
			return nil, nprotoErrorf(PayloadError, "assert input error: %s", err.Error())
		}

		w := &bytes.Buffer{}
		if err := client.encoder.EncodeData(w, input); err != nil {
			return nil, nprotoErrorf(PayloadError, "encode input error: %s", err.Error())
		}

		req := &nppb.NatsRPCRequest{
			Input: &nppb.RawData{
				EncoderName: client.encoder.EncoderName(),
				Bytes:       w.Bytes(),
			},
			MetaData: &nppb.MD{},
		}

		md := npmd.MDFromOutgoingContext(ctx)
		if md != nil {
			req.MetaData.From(md)
		}

		dl, ok := ctx.Deadline()
		if !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, client.timeout)
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
			return nil, nprotoErrorf(ProtocolError, "marshal request error: %s", err.Error())
		}

		respMsg, err := client.nc.RequestWithContext(
			ctx,
			subjectFormat(client.subjectPrefix, spec.SvcName, spec.MethodName),
			reqData,
		)
		if err != nil {
			return nil, err
		}

		resp := &nppb.NatsRPCResponse{}
		if err := proto.Unmarshal(respMsg.Data, resp); err != nil {
			return nil, nprotoErrorf(ProtocolError, "unmarshal response error: %s", err.Error())
		}

		switch out := resp.Out.(type) {
		case *nppb.NatsRPCResponse_Output:
			if out.Output.EncoderName != client.encoder.EncoderName() {
				return nil, nprotoErrorf(
					ProtocolError,
					"expect output encoded by %s, but got %s",
					client.encoder.EncoderName(),
					out.Output.EncoderName,
				)
			}

			output := spec.NewOutput()
			if err := client.encoder.DecodeData(bytes.NewReader(out.Output.Bytes), output); err != nil {
				return nil, nprotoErrorf(
					PayloadError,
					"decode output error: %s",
					err.Error(),
				)
			}

			return output, nil

		case *nppb.NatsRPCResponse_Err:
			return nil, out.Err.To()

		case *nppb.NatsRPCResponse_PlainErr:
			return nil, errors.New(out.PlainErr)

		default:
			panic(fmt.Errorf("Unexpected branch"))
		}

	}
}
