package binlogmsg

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// OptLogger sets structured logger.
func OptLogger(logger *zerolog.Logger) Option {
	return func(pipe *BinlogMsgPipe) error {
		pipe.logger = logger.With().Str("component", "nproto.npmsg.binlogmsg.BinlogMsgPipe").Logger()
		return nil
	}
}

// OptLockName sets the lock name used by 'SELECT GET_LOCK'.
func OptLockName(lockName string) Option {
	return func(pipe *BinlogMsgPipe) error {
		pipe.lockName = lockName
		return nil
	}
}

// OptMaxInflight sets the max number of messages not yet processed.
func OptMaxInflight(maxInflight int) Option {
	return func(pipe *BinlogMsgPipe) error {
		if maxInflight <= 0 {
			return fmt.Errorf("nproto.npmsg.binlogmsg.BinlogMsgPipe: MaxInflight must >= 1")
		}
		pipe.maxInflight = maxInflight
		return nil
	}
}

// OptRetryWait sets the interval between getting reconnecting db.
func OptRetryWait(retryWait time.Duration) Option {
	return func(pipe *BinlogMsgPipe) error {
		pipe.retryWait = retryWait
		return nil
	}
}
