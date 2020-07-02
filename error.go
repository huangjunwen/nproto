package nproto

import (
	"fmt"
)

// Error should be handled properly by all component (rpc/msg...) implementations.
type Error struct {
	// Code is the error code.
	Code ErrorCode `json:"code"`

	// Message is the description of the error.
	Message string `json:"message"`
}

// Errorf creates a new Error.
func Errorf(code ErrorCode, msg string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(msg, args...),
	}
}

// Error implements error interface.
func (err *Error) Error() string {
	sep := ""
	if err.Message != "" {
		sep = ": "
	}
	return fmt.Sprintf("nproto.Error(%d%s%s)", err.Code, sep, err.Message)
}

// ErrorCode describes the reason of Error.
//
// Range -32768 ~ -32000 are reserved (like json-rpc).
// Range -32768 ~ -32500 are not retryable error.
type ErrorCode int16

const (
	// ProtocolError should be returned when error occurs in protocol level, for example:
	//   - Parse request error.
	//   - ...
	ProtocolError ErrorCode = -32700

	// PayloadError should be returned when error occurs in payload level, for example:
	//   - Method not found in rpc.
	//   - Parameter can't be encoded/decoded correctly.
	//   - Parameter's value out of range.
	//   - ...
	PayloadError ErrorCode = -32600

	// NotRetryableError should be returned when you should not try again.
	NotRetryableError ErrorCode = -32500
)

// Retryable returns true when the code > -32500.
func (ec ErrorCode) Retryable() bool {
	return ec > -32500
}
