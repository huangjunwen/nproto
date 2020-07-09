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
	return fmt.Sprintf("nproto.Error(%s%s%s)", err.Code.String(), sep, err.Message)
}
