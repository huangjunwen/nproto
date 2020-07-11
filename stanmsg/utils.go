package stanmsg

import (
	"errors"
)

var (
	ErrNCMaxReconnect = errors.New("stanmsg.DurConn nats.Conn should have MaxReconnects < 0")

	ErrDupSubscription = errors.New("stanmsg.DurConn duplicated subscription")

	ErrDurConnClosed = errors.New("stanmsg.DurConn closed")
)
