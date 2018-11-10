package dbstore

import (
	"time"

	"github.com/rs/zerolog"
)

// OptLogger sets structured logger.
func OptLogger(logger *zerolog.Logger) Option {
	return func(store *DBMsgStore) error {
		store.logger = logger.With().Str("component", "nproto.npmsg.dbstore.DBMsgStore").Logger()
		return nil
	}
}

// OptRetryWait sets the interval between db reconnections and lock acquiring.
func OptRetryWait(t time.Duration) Option {
	return func(store *DBMsgStore) error {
		store.retryWait = t
		return nil
	}
}

// OptFlushWait sets the interval between flushing messages to downstream.
func OptFlushWait(t time.Duration) Option {
	return func(store *DBMsgStore) error {
		store.flushWait = t
		return nil
	}
}

// OptBatch sets the number of messages inflight (not yet ack) when flushing to downstream.
func OptBatch(batch int) Option {
	return func(store *DBMsgStore) error {
		store.batch = batch
		return nil
	}
}
