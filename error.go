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

// Error implements error interface.
func (err *Error) Error() string {
	return fmt.Sprintf("nproto.Error<%d: %s>", err.Code, err.Message)
}

// ErrorCode describe the reason of Error.
//
// Range -32768 ~ -32000 are reserved (like json-rpc).
// Range -32768 ~ -32500 are not retryable error.
type ErrorCode int16

const (
	// ParseError should be returned when requests can't be parsed correctly.
	ParseError ErrorCode = -32700

	// InvalidError should be returned when requests are well-formed but invalid, for example:
	//   - Method not found in rpc.
	//   - Payload/parameter can't be parsed correctly or value invalid.
	//   - ...
	InvalidError ErrorCode = -32600

	// NotRetryableError should be returned when the requests can't be retried due to other reason.
	NotRetryableError ErrorCode = -32500
)

// Retryable returns true when the code > -32500.
func (ec ErrorCode) Retryable() bool {
	return ec > -32500
}
