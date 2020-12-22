package binlogmsg

import (
	"fmt"
	"time"

	"github.com/huangjunwen/golibs/logr"
)

var (
	// DefaultLockName is the default value of PipeOptLockName.
	DefaultLockName = "nproto.binlogmsg"

	// DefaultMaxInflight is the default value of PipeOptMaxInflight.
	DefaultMaxInflight = 4096

	// DefaultRetryWait is the default value of PipeOptRetryWait.
	DefaultRetryWait = 5 * time.Second
)

// PipeOptLogger sets logger for MsgPipe.
func PipeOptLogger(logger logr.Logger) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if logger == nil {
			logger = logr.Nop
		}
		pipe.logger = logger.WithValues("component", "nproto.binlogmsg.MsgPipe")
		return nil
	}
}

// PipeOptLockName sets the lock name using in MySQL get lock ("SELECT GET_LOCK"):
// only one instance of pipes can run for the same lock name.
func PipeOptLockName(lockName string) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if lockName == "" {
			return fmt.Errorf("PipeOptLockName got empty lockName")
		}
		pipe.lockName = lockName
		return nil
	}
}

// PipeOptLockPingInterval sets the ping interval for mysql GET_LOCK connection.
func PipeOptLockPingInterval(pingInterval time.Duration) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if pingInterval < 0 {
			return fmt.Errorf("PipeOptLockPingInterval < 0")
		}
		pipe.lockPingInterval = pingInterval
		return nil
	}
}

// PipeOptLockCooldownInterval sets the time after mysql GET_LOCK and before
// actual work.
func PipeOptLockCooldownInterval(cooldownInterval time.Duration) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if cooldownInterval < 0 {
			return fmt.Errorf("PipeOptLockCooldownInterval < 0")
		}
		pipe.lockCooldownInterval = cooldownInterval
		return nil
	}
}

// PipeOptMaxInflight sets the max number of messages inflight (publishing).
func PipeOptMaxInflight(maxInflight int) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if maxInflight < 1 {
			return fmt.Errorf("PipeOptMaxInflight should be at least 1, but got %d", maxInflight)
		}
		pipe.maxInflight = maxInflight
		return nil
	}
}

// PipeOptRetryWait sets the interval between retries due to all kinds of errors.
func PipeOptRetryWait(t time.Duration) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if t <= 0 {
			return fmt.Errorf("PipeOptRetryWait got non-positive duration %s", t.String())
		}
		pipe.retryWait = t
		return nil
	}
}
