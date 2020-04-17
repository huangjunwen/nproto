package fulldump

import (
	"context"

	"github.com/pkg/errors"

	. "github.com/huangjunwen/nproto/helpers/mycanal"
)

// FullDump is similar to  mysqldump --single-transaction, see mycanal's doc for prerequisites.
//
// NOTE: Since `BeginTx` can not run 'START TRANSACTION WITH CONSISTENT SNAPSHOT', use `*sql.Conn` instead of `*sql.Tx`.
// More discussions:
//   - https://github.com/golang/go/issues/19981
//
// Other refs:
//   - https://issues.redhat.com/browse/DBZ-210
func FullDump(
	ctx context.Context,
	cfg *FullDumpConfig,
	handler Handler,
) (gtidSet string, err error) {

	// Some commands does not need cancel.
	bgCtx := context.Background()

	db, err := cfg.Client()
	if err != nil {
		return "", errors.WithMessage(err, "fulldump.FullDump open client error")
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		return "", errors.WithMessage(err, "fulldump.FullDump open conn error")
	}
	defer conn.Close()

	// 0. Set isolation level to repeatable read (the default).
	if _, err = conn.ExecContext(bgCtx, "SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ"); err != nil {
		return "", errors.WithMessage(err, "fulldump.FullDump set isolation level error")
	}

	// 1. Lock tables: to get current GTID set and start trx.
	// NOTE: the lock will be released if connection closed.
	_, err = conn.ExecContext(bgCtx, "FLUSH TABLES WITH READ LOCK")
	if err != nil {
		return "", errors.WithMessage(err, "fulldump.FullDump ftwrl error")
	}
	defer func() {
		// XXX: to ensure unlock is run
		conn.ExecContext(bgCtx, "UNLOCK TABLES")
	}()

	// 2. Start trx with consistent snapshot.
	if _, err = conn.ExecContext(bgCtx, "START TRANSACTION WITH CONSISTENT SNAPSHOT"); err != nil {
		return "", errors.WithMessage(err, "fulldump.FullDump start transaction error")
	}
	defer func() {
		// fulldump should not modify data
		conn.ExecContext(bgCtx, "ROLLBACK")
	}()

	// 3. Get GTID_EXECUTED
	err = conn.QueryRowContext(bgCtx, "SELECT @@GLOBAL.GTID_EXECUTED").Scan(&gtidSet)
	if err != nil {
		return "", errors.WithMessage(err, "fulldump.FullDump get gtid error")
	}
	if gtidSet == "" {
		return "", errors.Errorf("No GTID_EXECUTED, pls make sure you have turn on binlog and gtid mode")
	}

	// 4. Unlock tables.
	_, err = conn.ExecContext(bgCtx, "UNLOCK TABLES")
	if err != nil {
		return "", errors.WithMessage(err, "fulldump.FullDump unlock tables error")
	}

	// 5. User function
	err = handler(ctx, conn)
	if err != nil {
		return "", err
	}

	return
}
