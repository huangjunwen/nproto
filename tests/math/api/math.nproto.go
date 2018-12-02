// Code generated by protoc-gen-nproto. DO NOT EDIT.
// source: math.proto

package mathapi // import "github.com/huangjunwen/nproto/tests/math/api"

import (
	context "context"
	proto "github.com/golang/protobuf/proto"
	nproto "github.com/huangjunwen/nproto/nproto"
)

// Avoid import and not used errors.
var (
	_ = context.Background
	_ = proto.Int
	_ = nproto.NewRPCCtx
)

// @@protoc_insertion_point(imports)

//Math is a service providing some math functions.
// @@nprpc@@
type Math interface {

	//Sum returns the sum of a list of arguments.
	Sum(ctx context.Context, input *SumRequest) (output *SumReply, err error)
}

// ServeMath serves Math service using a RPC server.
func ServeMath(server nproto.RPCServer, svcName string, svc Math) error {
	return server.RegistSvc(svcName, map[*nproto.RPCMethod]nproto.RPCHandler{
		methodMath__Sum: func(ctx context.Context, input proto.Message) (proto.Message, error) {
			return svc.Sum(ctx, input.(*SumRequest))
		},
	})
}

// InvokeMath invokes Math service using a RPC client.
func InvokeMath(client nproto.RPCClient, svcName string) Math {
	return &clientMath{
		handlerSum: client.MakeHandler(svcName, methodMath__Sum),
	}
}

var methodMath__Sum = &nproto.RPCMethod{
	Name:      "Sum",
	NewInput:  func() proto.Message { return &SumRequest{} },
	NewOutput: func() proto.Message { return &SumReply{} },
}

type clientMath struct {
	handlerSum nproto.RPCHandler
}

// Sum implements Math interface.
func (svc *clientMath) Sum(ctx context.Context, input *SumRequest) (*SumReply, error) {
	output, err := svc.handlerSum(ctx, input)
	if err != nil {
		return nil, err
	}
	return output.(*SumReply), nil
}

// @@protoc_insertion_point(nprpc)

// @@protoc_insertion_point(npmsg)
