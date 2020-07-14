package nppbrpc

import (
	nprpc "github.com/huangjunwen/nproto/v2/rpc"
)

// To writes to *nprpc.RPCError
func (e *RPCError) To() *nprpc.RPCError {
	return &nprpc.RPCError{
		Code:    nprpc.RPCErrorCode(e.Code),
		Message: e.Message,
	}
}

// From reads from src.
func (e *RPCError) From(src *nprpc.RPCError) {
	e.Code = int32(src.Code)
	e.Message = src.Message
}
