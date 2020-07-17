package stanmsg

import (
	"errors"
)

var (
	// ErrNCMaxReconnect is returned if nc has MaxReconnects >= 0.
	ErrNCMaxReconnect = errors.New("stanmsg.DurConn nats.Conn should have MaxReconnects < 0")

	// ErrClosed is returned when DurConn is closed
	ErrClosed = errors.New("stanmsg.DurConn closed")

	// ErrDupSubscription is returned when (subjectName, queue) already subscribed.
	ErrDupSubscription = errors.New("stanmsg.DurConn duplicated subscription")

	// ErrNotConnected is returned when stan connection has not connected yet.
	ErrNotConnected = errors.New("stanmsg.DurConn not connected")
)
