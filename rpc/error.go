package rpc

//go:generate stringer -type=RPCErrorCode

import (
	"fmt"
)

// RPCError should be handled properly by all rpc implementations.
type RPCError struct {
	Code    RPCErrorCode `json:"code"`
	Message string       `json:"message"`
}

// RPCErrorCode describes the reason of rpc error.
// Negative (<0) are reserved for pre defined error code.
// Positive (>0) are reserved for user defined error code.
type RPCErrorCode int16

// RPCErrorf creates an new RPCError.
func RPCErrorf(code RPCErrorCode, msg string, args ...interface{}) *RPCError {
	return &RPCError{
		Code:    code,
		Message: fmt.Sprintf(msg, args...),
	}
}

// Error implements error interface.
func (err *RPCError) Error() string {
	sep := ""
	if err.Message != "" {
		sep = " "
	}
	return fmt.Sprintf("RPCError(%s%s%s)", err.Code, sep, err.Message)
}

// Pre-defined rpc error codes.
const (
	UnknownError RPCErrorCode = -32600 - iota
	SvcNotFound
	MethodNotFound
	EncodeRequestError
	DecodeRequestError
	EncodeResponseError
	DecodeResponseError
)
