package zlog

import (
	"os"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	// DefaultZLogger is the default zerolog.Logger used by nproto packages.
	// If os.Stdout is a terminal then ConsoleWriter will be used for prettier output.
	// You can override this to whatever you want to log to.
	DefaultZLogger = zerolog.New(os.Stdout).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Logger()
)

func init() {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		DefaultZLogger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			Level(zerolog.InfoLevel).
			With().
			Timestamp().
			Logger()
	}
}
