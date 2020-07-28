package binlogmsg

import (
	"fmt"
	"time"

	"github.com/huangjunwen/golibs/logr"
)

var (
	DefaultLockName = "nproto.binlogmsg"

	DefaultMaxInflight = 4096

	DefaultRetryWait = 5 * time.Second
)

func PipeOptLogger(logger logr.Logger) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if logger == nil {
			logger = logr.Nop
		}
		pipe.logger = logger.WithValues("component", "nproto.binlogmsg.MsgPipe")
		return nil
	}
}

func PipeOptLockName(lockName string) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if lockName == "" {
			return fmt.Errorf("PipeOptLockName got empty lockName")
		}
		pipe.lockName = lockName
		return nil
	}
}

func PipeOptMaxInflight(maxInflight int) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if maxInflight < 1 {
			return fmt.Errorf("PipeOptMaxInflight should be at least 1, but got %d", maxInflight)
		}
		pipe.maxInflight = maxInflight
		return nil
	}
}

func PipeOptRetryWait(t time.Duration) MsgPipeOption {
	return func(pipe *MsgPipe) error {
		if t <= 0 {
			return fmt.Errorf("PipeOptRetryWait got non-positive duration %s", t.String())
		}
		pipe.retryWait = t
		return nil
	}
}
