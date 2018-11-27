package dbstore

import (
	"fmt"

	"github.com/rs/zerolog"
)

// OptLogger sets structured logger.
func OptLogger(logger *zerolog.Logger) Option {
	return func(store *DBStore) error {
		store.logger = logger.With().Str("component", "nproto.npmsg.dbstore.DBStore").Logger()
		return nil
	}
}

func OptDeleteBulkSize(bulkSize int) Option {
	return func(store *DBStore) error {
		if bulkSize <= 0 {
			return fmt.Errorf("nproto.npmsg.dbstore.DBStore: DeleteBulkSize must >= 1")
		}
		store.deleteBulkSize = bulkSize
		return nil
	}
}

func OptMaxInflight(maxInflight int) Option {
	return func(store *DBStore) error {
		if maxInflight <= 0 {
			return fmt.Errorf("nproto.npmsg.dbstore.DBStore: MaxInflight must >= 1")
		}
		store.maxInflight = maxInflight
		return nil
	}
}

func OptMaxBuf(maxBuf int) Option {
	return func(store *DBStore) error {
		if maxBuf <= 0 {
			return fmt.Errorf("nproto.npmsg.dbstore.DBStore: MaxBuf must >= 1")
		}
		store.maxBuf = maxBuf
		return nil
	}
}

func OptCreateTable() Option {
	return func(store *DBStore) error {
		store.createTable = true
		return nil
	}
}
