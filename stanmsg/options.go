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
	DefaultSubjectPrefix = "stanmsg"

	DefaultReconnectWait = 5 * time.Second

	DefaultStanPingInterval = stan.DefaultPingInterval

	DefaultStanPingMaxOut = stan.DefaultPingMaxOut

	DefaultStanPubAckWait = 2 * time.Second

	DefaultSubRetryWait = 5 * time.Second
)

var (
	subjectPrefixRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func DCOptLogger(logger logr.Logger) DurConnOption {
	return func(dc *DurConn) error {
		if logger == nil {
			logger = logr.Nop
		}
		dc.logger = logger.WithValues("component", "nproto.stanmsg.DurConn")
		return nil
	}
}

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

func DCOptSubjectPrefix(subjectPrefix string) DurConnOption {
	return func(dc *DurConn) error {
		if !subjectPrefixRegexp.MatchString(subjectPrefix) {
			return fmt.Errorf("DCOptSubjectPrefix got invalid subject prefix %q", subjectPrefix)
		}
		dc.subjectPrefix = subjectPrefix
		return nil
	}
}

func DCOptContext(ctx context.Context) DurConnOption {
	return func(dc *DurConn) error {
		if ctx == nil {
			return fmt.Errorf("DCOptContext got nil context.Context")
		}
		dc.ctx = ctx
		return nil
	}
}

func DCOptReconnectWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		if t <= 0 {
			return fmt.Errorf("DCOptReconnectWait got non-positive duration %s", t.String())
		}
		dc.reconnectWait = t
		return nil
	}
}

func DCOptSubRetryWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		if t <= 0 {
			return fmt.Errorf("DCOptSubRetryWait got non-positive duration %s", t.String())
		}
		dc.subRetryWait = t
		return nil
	}
}

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

func DCOptStanPubAckWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		if t <= 0 {
			return fmt.Errorf("DCOptStanPubAckWait got non-positive duration %s", t.String())
		}
		dc.stanOptPubAckWait = t
		return nil
	}
}

func DCOptConnectCb(cb func(stan.Conn)) DurConnOption {
	return func(dc *DurConn) error {
		if cb == nil {
			cb = func(_ stan.Conn) {}
		}
		dc.connectCb = cb
		return nil
	}
}

func DCOptDisconnectCb(cb func(stan.Conn)) DurConnOption {
	return func(dc *DurConn) error {
		if cb == nil {
			cb = func(_ stan.Conn) {}
		}
		dc.disconnectCb = cb
		return nil
	}
}

func DCOptSubscribeCb(cb func(stan.Conn, MsgSpec)) DurConnOption {
	return func(dc *DurConn) error {
		if cb == nil {
			cb = func(_ stan.Conn, _ MsgSpec) {}
		}
		dc.subscribeCb = cb
		return nil
	}
}

func SubOptStanAckWait(t time.Duration) SubOption {
	return func(sub *subscription) error {
		if t <= 0 {
			return fmt.Errorf("SubOptStanAckWait got non-positive duration %s", t.String())
		}
		sub.stanOptions = append(sub.stanOptions, stan.AckWait(t))
		return nil
	}
}
