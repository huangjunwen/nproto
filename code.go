//go:generate stringer -type=ErrorCode
package nproto

// ErrorCode describes the reason of Error.
type ErrorCode int16

const (
	/* RPC error range [-32600 ~ -32699] */

	// RPCError is general error for rpc.
	RPCError ErrorCode = -32600 - iota

	// RPCSpecInvalid should be returned if RPCSpec is not valid.
	RPCSpecInvalid

	// RPCSvcNotFound should be returned when service not found.
	RPCSvcNotFound

	// RPCMethodNotFound should be returned when method not found.
	RPCMethodNotFound

	// RPCRequestEncodeError should be returned when rpc request can't be encoded.
	RPCRequestEncodeError

	// RPCRequestDecodeError should be returned when rpc request can't be decoded.
	RPCRequestDecodeError

	// RPCResponseEncodeError should be returned when rpc response can't be encoded.
	RPCResponseEncodeError

	// RPCResponseDecodeError should be returned when rpc response can't be decoded.
	RPCResponseDecodeError
)
