package natsrpc

import (
	"errors"
)

var (
	// ErrNCMaxReconnect is returned if nc has MaxReconnects >= 0.
	ErrNCMaxReconnect = errors.New("natsrpc.Server nats.Conn should have MaxReconnects < 0")

	// ErrServerClosed is returned when the server is closed
	ErrServerClosed = errors.New("natsrpc.Server closed")
)
