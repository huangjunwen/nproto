package durconn

import (
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/rs/zerolog"
)

// OptLogger sets structured logger.
func OptLogger(logger *zerolog.Logger) Option {
	return func(dc *DurConn) error {
		dc.logger = logger.With().Str("component", "nproto.npmsg.durconn.DurConn").Logger()
		return nil
	}
}

// OptReconnectWait sets reconnection wait, e.g. time between reconnections.
func OptReconnectWait(t time.Duration) Option {
	return func(dc *DurConn) error {
		dc.reconnectWait = t
		return nil
	}
}

// OptSubjectPrefix sets message subject prefix.
// Default "npmsg": If you publish a message with subject "xxx", then the actual subject is "npmsg.xxx".
func OptSubjectPrefix(prefix string) Option {
	return func(dc *DurConn) error {
		dc.setSubjectPrefix(prefix)
		return nil
	}
}

// OptConnectCb sets a callback invoked each time a new stan.Conn is established.
func OptConnectCb(fn func(sc stan.Conn)) Option {
	return func(dc *DurConn) error {
		dc.connectCb = fn
		return nil
	}
}

// OptDisconnectCb sets a callback invoked each time a stan.Conn lost.
func OptDisconnectCb(fn func(sc stan.Conn)) Option {
	return func(dc *DurConn) error {
		dc.disconnectCb = fn
		return nil
	}
}

// OptPings sets stan.Pings.
func OptPings(interval, maxOut int) Option {
	return func(dc *DurConn) error {
		dc.stanOptions = append(dc.stanOptions, stan.Pings(interval, maxOut))
		return nil
	}
}

// OptPubAckWait sets stan.PubAckWait.
func OptPubAckWait(t time.Duration) Option {
	return func(dc *DurConn) error {
		dc.stanOptions = append(dc.stanOptions, stan.PubAckWait(t))
		return nil
	}
}

// SubOptRetryWait sets the wait time between subscription.
func SubOptRetryWait(t time.Duration) SubOption {
	return func(sub *subscription) error {
		sub.retryWait = t
		return nil
	}
}

// SubOptSubscribeCb sets a callback invoked each time a subscription is established.
// NOTE: The callback is also called when resubscribing after reconnection.
func SubOptSubscribeCb(fn func(sc stan.Conn, subject, queue string)) SubOption {
	return func(sub *subscription) error {
		sub.subscribeCb = fn
		return nil
	}
}

// SubOptAckWait sets stan.AckWait.
func SubOptAckWait(t time.Duration) SubOption {
	return func(sub *subscription) error {
		sub.stanOptions = append(sub.stanOptions, stan.AckWait(t))
		return nil
	}
}
