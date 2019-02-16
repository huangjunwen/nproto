package dbpipe

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// OptLogger sets structured logger.
func OptLogger(logger *zerolog.Logger) Option {
	return func(pipe *DBMsgPublisherPipe) error {
		pipe.logger = logger.With().Str("component", "nproto.npmsg.dbpipe.DBMsgPublisherPipe").Logger()
		return nil
	}
}

// OptMaxDelBulkSz sets the max number of messages to delete each time after delivered to downstream.
func OptMaxDelBulkSz(bulkSize int) Option {
	return func(pipe *DBMsgPublisherPipe) error {
		if bulkSize <= 0 {
			return fmt.Errorf("nproto.npmsg.dbpipe.DBMsgPublisherPipe: DeleteBulkSize must >= 1")
		}
		pipe.maxDelBulkSz = bulkSize
		return nil
	}
}

// OptMaxInflight sets the max number of messages to publish to downstream before acknowledgement.
func OptMaxInflight(maxInflight int) Option {
	return func(pipe *DBMsgPublisherPipe) error {
		if maxInflight <= 0 {
			return fmt.Errorf("nproto.npmsg.dbpipe.DBMsgPublisherPipe: MaxInflight must >= 1")
		}
		pipe.maxInflight = maxInflight
		return nil
	}
}

// OptMaxBuf sets the max number of messages buffered in memory.
func OptMaxBuf(maxBuf int) Option {
	return func(pipe *DBMsgPublisherPipe) error {
		if maxBuf < 0 {
			return fmt.Errorf("nproto.npmsg.dbpipe.DBMsgPublisherPipe: MaxBuf must >= 0")
		}
		pipe.maxBuf = maxBuf
		return nil
	}
}

// OptRetryWait sets the interval between getting db connection and lock.
func OptRetryWait(t time.Duration) Option {
	return func(pipe *DBMsgPublisherPipe) error {
		pipe.retryWait = t
		return nil
	}
}

// OptFlushWait sets the interval between redelivery.
func OptFlushWait(t time.Duration) Option {
	return func(pipe *DBMsgPublisherPipe) error {
		pipe.flushWait = t
		return nil
	}
}

// OptNoRedeliveryLoop prevent DBMsgPublisherPipe to launch the redelivery loop.
func OptNoRedeliveryLoop() Option {
	return func(pipe *DBMsgPublisherPipe) error {
		pipe.noRedeliveryLoop = true
		return nil
	}
}
