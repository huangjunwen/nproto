package stanmsg

import (
	"context"
	"fmt"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/taskrunner"
	"github.com/nats-io/stan.go"
)

var (
	DefaultSubjectPrefix = "stanmsg"

	DefaultReconnectWait = 5 * time.Second

	DefaultStanPingInterval = stan.DefaultPingInterval

	DefaultStanPingMaxOut = stan.DefaultPingMaxOut

	DefaultStanPubAckWait = 2 * time.Second

	DefaultSubRetryWait = 5 * time.Second
)

func DCOptLogger(logger logr.Logger) DurConnOption {
	return func(dc *DurConn) error {
		dc.logger = logger.WithValues("component", "nproto.stanmsg.DurConn")
		return nil
	}
}

func DCOptRunner(runner taskrunner.TaskRunner) DurConnOption {
	return func(dc *DurConn) error {
		dc.runner.Close()
		dc.runner = runner
		return nil
	}
}

func DCOptSubjectPrefix(subjectPrefix string) DurConnOption {
	return func(dc *DurConn) error {
		dc.subjectPrefix = subjectPrefix
		return nil
	}
}

func DCOptContext(ctx context.Context) DurConnOption {
	return func(dc *DurConn) error {
		dc.ctx = ctx
		return nil
	}
}

func DCOptReconnectWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		dc.reconnectWait = t
		return nil
	}
}

func DCOptSubRetryWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		dc.subRetryWait = t
		return nil
	}
}

func DCOptStanPingInterval(interval int) DurConnOption {
	return func(dc *DurConn) error {
		if interval < 1 {
			// See: stan.go::Pings
			return fmt.Errorf("DCOptStanPingInterval(%d) < 1", interval)
		}
		dc.stanOptPingInterval = interval
		return nil
	}
}

func DCOptStanPingMaxOut(maxOut int) DurConnOption {
	return func(dc *DurConn) error {
		if maxOut < 2 {
			// See: stan.go::Pings
			return fmt.Errorf("DCOptStanPingMaxOut(%d) < 2", maxOut)
		}
		dc.stanOptPingMaxOut = maxOut
		return nil
	}
}

func DCOptStanPubAckWait(t time.Duration) DurConnOption {
	return func(dc *DurConn) error {
		dc.stanOptPubAckWait = t
		return nil
	}
}

func DCOptConnectCb(cb func(stan.Conn)) DurConnOption {
	return func(dc *DurConn) error {
		dc.connectCb = cb
		return nil
	}
}

func DCOptDisconnectCb(cb func(stan.Conn)) DurConnOption {
	return func(dc *DurConn) error {
		dc.disconnectCb = cb
		return nil
	}
}

func SubOptStanAckWait(t time.Duration) SubOption {
	return func(sub *subscription) error {
		sub.stanOptions = append(sub.stanOptions, stan.AckWait(t))
		return nil
	}
}
