package zlog

import (
	"os"

	"github.com/rs/zerolog"
)

var (
	// DefaultZLogger is the default zero logger used by nproto packages.
	DefaultZLogger = zerolog.New(os.Stdout).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Logger()
)
