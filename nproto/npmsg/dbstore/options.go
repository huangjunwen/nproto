package dbstore

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// OptLogger sets structured logger.
func OptLogger(logger *zerolog.Logger) Option {
	return func(store *DBStore) error {
		store.logger = logger.With().Str("component", "nproto.npmsg.dbstore.DBStore").Logger()
		return nil
	}
}

// OptMaxDelBulkSz sets the max number of messages to delete each time after delivered to downstream.
func OptMaxDelBulkSz(bulkSize int) Option {
	return func(store *DBStore) error {
		if bulkSize <= 0 {
			return fmt.Errorf("nproto.npmsg.dbstore.DBStore: DeleteBulkSize must >= 1")
		}
		store.maxDelBulkSz = bulkSize
		return nil
	}
}

// OptMaxInflight sets the max number of messages to publish to downstream before acknowledgement.
func OptMaxInflight(maxInflight int) Option {
	return func(store *DBStore) error {
		if maxInflight <= 0 {
			return fmt.Errorf("nproto.npmsg.dbstore.DBStore: MaxInflight must >= 1")
		}
		store.maxInflight = maxInflight
		return nil
	}
}

// OptMaxBuf sets the max number of messages buffered in memory.
func OptMaxBuf(maxBuf int) Option {
	return func(store *DBStore) error {
		if maxBuf < 0 {
			return fmt.Errorf("nproto.npmsg.dbstore.DBStore: MaxBuf must >= 0")
		}
		store.maxBuf = maxBuf
		return nil
	}
}

// OptRetryWait sets the interval between getting db connection and lock.
func OptRetryWait(t time.Duration) Option {
	return func(store *DBStore) error {
		store.retryWait = t
		return nil
	}
}

// OptFlushWait sets the interval between redelivery.
func OptFlushWait(t time.Duration) Option {
	return func(store *DBStore) error {
		store.flushWait = t
		return nil
	}
}

// OptNoRedeliveryLoop prevent DBStore to launch the redelivery loop.
func OptNoRedeliveryLoop() Option {
	return func(store *DBStore) error {
		store.noRedeliveryLoop = true
		return nil
	}
}
