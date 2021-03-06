package binlogmsg

import (
	"fmt"
	"time"

	"github.com/huangjunwen/golibs/logr"
)

var (
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
