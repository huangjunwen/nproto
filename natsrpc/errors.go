package natsrpc

import (
	"errors"
)

var (
	// ErrNCMaxReconnect is returned if nats.Conn has MaxReconnects >= 0.
	ErrNCMaxReconnect = errors.New("natsrpc.ServerConn nats.Conn should have MaxReconnects < 0")

	// ErrClosed is returned when ServerConn is closed
	ErrClosed = errors.New("natsrpc.ServerConn closed")
)
