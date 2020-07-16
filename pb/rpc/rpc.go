package nppbrpc

import (
	nprpc "github.com/huangjunwen/nproto/v2/rpc"
)

// To converts to *nprpc.RPCError
func (e *RPCError) To() *nprpc.RPCError {
	return &nprpc.RPCError{
		Code:    nprpc.RPCErrorCode(e.Code),
		Message: e.Message,
	}
}

// NewRPCError converts from *nprpc.RPCError.
func NewRPCError(src *nprpc.RPCError) *RPCError {
	return &RPCError{
		Code:    int32(src.Code),
		Message: src.Message,
	}
}
