package fulldump

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

// FullDump is similar to  mysqldump --single-transaction, see mycanal's doc for prerequisites.
//
// NOTE: Since `BeginTx` can not run 'START TRANSACTION WITH CONSISTENT SNAPSHOT', use `*sql.Conn` instead of `*sql.Tx`.
// More discussions:
//   - https://github.com/golang/go/issues/19981
//
// Other refs:
//   - https://issues.redhat.com/browse/DBZ-210
func FullDump(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Conn) error) (gtidSet string, err error) {
	// 0. Use a single connection within this whole function.
	conn, err := db.Conn(ctx)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer conn.Close()

	// 1. Lock tables: to get current GTID set and start trx.
	_, err = conn.ExecContext(ctx, "FLUSH TABLES WITH READ LOCK")
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer func() {
		// XXX: to ensure unlock is run
		conn.ExecContext(ctx, "UNLOCK TABLES")
	}()

	// 2. Start trx with consistent snapshot.
	if _, err = conn.ExecContext(ctx, "SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ"); err != nil {
		return "", errors.WithStack(err)
	}
	if _, err = conn.ExecContext(ctx, "START TRANSACTION WITH CONSISTENT SNAPSHOT"); err != nil {
		return "", errors.WithStack(err)
	}
	defer func() {
		// fulldump should not modify data
		conn.ExecContext(ctx, "ROLLBACK")
	}()

	// 3. Get GTID_EXECUTED
	err = conn.QueryRowContext(ctx, "SELECT @@GLOBAL.GTID_EXECUTED").Scan(&gtidSet)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if gtidSet == "" {
		return "", errors.Errorf("No GTID_EXECUTED, pls make sure you have turn on gtid mode")
	}

	// 4. Unlock tables.
	_, err = conn.ExecContext(ctx, "UNLOCK TABLES")
	if err != nil {
		return "", errors.WithStack(err)
	}

	// 5. User function
	err = fn(ctx, conn)
	if err != nil {
		return "", err
	}

	return
}
