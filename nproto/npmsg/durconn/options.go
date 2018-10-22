package durconn

import (
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/rs/zerolog"
)

// DurConnOptLogger sets structured logger.
func DurConnOptLogger(logger *zerolog.Logger) DurConnOption {
	return func(dc *DurConn) error {
		dc.logger = logger.With().Str("component", "nproto.npmsg.durconn.DurConn").Logger()
		return nil
	}
}

// DurConnOptReconnectWait sets reconnection wait, e.g. time between reconnections.
func DurConnOptReconnectWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		dc.reconnectWait = t
		return nil
	}
}

// DurConnOptSubjectPrefix sets message subject prefix.
// Default "npmsg": If you publish a message with subject "xxx", then the actual subject is "npmsg.xxx".
func DurConnOptSubjectPrefix(prefix string) DurConnOption {
	return func(dc *DurConn) error {
		dc.setSubjectPrefix(prefix)
		return nil
	}
}

// DurConnOptConnectCb sets a callback invoked each time a new stan.Conn is established.
func DurConnOptConnectCb(fn func(sc stan.Conn)) DurConnOption {
	return func(dc *DurConn) error {
		dc.connectCb = fn
		return nil
	}
}

// DurConnOptDisconnectCb sets a callback invoked each time a stan.Conn lost.
func DurConnOptDisconnectCb(fn func(sc stan.Conn)) DurConnOption {
	return func(dc *DurConn) error {
		dc.disconnectCb = fn
		return nil
	}
}

// DurConnOptPings sets stan.Pings.
func DurConnOptPings(interval, maxOut int) DurConnOption {
	return func(dc *DurConn) error {
		dc.stanOptions = append(dc.stanOptions, stan.Pings(interval, maxOut))
		return nil
	}
}

// DurConnOptPubAckWait sets stan.PubAckWait.
func DurConnOptPubAckWait(t time.Duration) DurConnOption {
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
