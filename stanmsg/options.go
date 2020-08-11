package stanmsg

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/taskrunner"
	"github.com/nats-io/stan.go"

	. "github.com/huangjunwen/nproto/v2/msg"
)

var (
	// DefaultSubjectPrefix is the default value of DCOptSubjectPrefix.
	DefaultSubjectPrefix = "stanmsg"

	// DefaultReconnectWait is the default value of DCOptReconnectWait.
	DefaultReconnectWait = 5 * time.Second

	// DefaultStanPingInterval is the default value of DCOptStanPingInterval.
	DefaultStanPingInterval = stan.DefaultPingInterval

	// DefaultStanPingMaxOut is the default value of DCOptStanPingMaxOut.
	DefaultStanPingMaxOut = stan.DefaultPingMaxOut

	// DefaultStanPubAckWait is the default value of DCOptStanPubAckWait.
	DefaultStanPubAckWait = 2 * time.Second

	// DefaultSubRetryWait is the default value of DCOptSubRetryWait.
	DefaultSubRetryWait = 5 * time.Second
)

var (
	subjectPrefixRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// DCOptLogger sets logger for DurConn.
func DCOptLogger(logger logr.Logger) DurConnOption {
	return func(dc *DurConn) error {
		if logger == nil {
			logger = logr.Nop
		}
		dc.logger = logger.WithValues("component", "nproto.stanmsg.DurConn")
		return nil
	}
}

// DCOptRunner sets runner for handlers.
func DCOptRunner(runner taskrunner.TaskRunner) DurConnOption {
	return func(dc *DurConn) error {
		if runner == nil {
			return fmt.Errorf("DCOptRunner got nil taskrunner.TaskRunner")
		}
		dc.runner.Close()
		dc.runner = runner
		return nil
	}
}

// DCOptSubjectPrefix sets subject prefix in nats streaming namespace.
func DCOptSubjectPrefix(subjectPrefix string) DurConnOption {
	return func(dc *DurConn) error {
		if !subjectPrefixRegexp.MatchString(subjectPrefix) {
			return fmt.Errorf("DCOptSubjectPrefix got invalid subject prefix %q", subjectPrefix)
		}
		dc.subjectPrefix = subjectPrefix
		return nil
	}
}

// DCOptContext sets base context for handlers.
func DCOptContext(ctx context.Context) DurConnOption {
	return func(dc *DurConn) error {
		if ctx == nil {
			return fmt.Errorf("DCOptContext got nil context.Context")
		}
		dc.ctx = ctx
		return nil
	}
}

// DCOptReconnectWait sets the interval between reconnections.
func DCOptReconnectWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		if t <= 0 {
			return fmt.Errorf("DCOptReconnectWait got non-positive duration %s", t.String())
		}
		dc.reconnectWait = t
		return nil
	}
}

// DCOptSubRetryWait sets the interval between resubscriptions due to subscription error.
func DCOptSubRetryWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		if t <= 0 {
			return fmt.Errorf("DCOptSubRetryWait got non-positive duration %s", t.String())
		}
		dc.subRetryWait = t
		return nil
	}
}

// DCOptStanPingInterval sets stan::Pings, must >= 1 (seconds).
func DCOptStanPingInterval(interval int) DurConnOption {
	return func(dc *DurConn) error {
		if interval < 1 {
			// See: stan.go::Pings
			return fmt.Errorf("DCOptStanPingInterval got too small interval (%d<1)", interval)
		}
		dc.stanOptPingInterval = interval
		return nil
	}
}

// DCOptStanPingMaxOut sets stan::Pings, must >= 2.
func DCOptStanPingMaxOut(maxOut int) DurConnOption {
	return func(dc *DurConn) error {
		if maxOut < 2 {
			// See: stan.go::Pings
			return fmt.Errorf("DCOptStanPingMaxOut got too small max out (%d<2)", maxOut)
		}
		dc.stanOptPingMaxOut = maxOut
		return nil
	}
}

// DCOptStanPubAckWait sets stan::PubAckWait.
func DCOptStanPubAckWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		if t <= 0 {
			return fmt.Errorf("DCOptStanPubAckWait got non-positive duration %s", t.String())
		}
		dc.stanOptPubAckWait = t
		return nil
	}
}

// DCOptConnectCb sets callback when nats streaming connection establish.
func DCOptConnectCb(cb func(stan.Conn)) DurConnOption {
	return func(dc *DurConn) error {
		if cb == nil {
			cb = func(_ stan.Conn) {}
		}
		dc.connectCb = cb
		return nil
	}
}

// DCOptDisconnectCb sets callback when nats streaming connection lost due to unexpected errors.
func DCOptDisconnectCb(cb func(stan.Conn)) DurConnOption {
	return func(dc *DurConn) error {
		if cb == nil {
			cb = func(_ stan.Conn) {}
		}
		dc.disconnectCb = cb
		return nil
	}
}

// DCOptSubscribeCb sets callback when subscriptions subscribed.
func DCOptSubscribeCb(cb func(stan.Conn, MsgSpec)) DurConnOption {
	return func(dc *DurConn) error {
		if cb == nil {
			cb = func(_ stan.Conn, _ MsgSpec) {}
		}
		dc.subscribeCb = cb
		return nil
	}
}

// SubOptStanAckWait sets stan::AckWait for a subscription.
func SubOptStanAckWait(t time.Duration) SubOption {
	return func(sub *subscription) error {
		if t <= 0 {
			return fmt.Errorf("SubOptStanAckWait got non-positive duration %s", t.String())
		}
		sub.stanOptions = append(sub.stanOptions, stan.AckWait(t))
		return nil
	}
}
