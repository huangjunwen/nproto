package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/huangjunwen/nproto/npmsg/dbstore"

	_ "github.com/go-sql-driver/mysql"
)

type nxDialect struct{}

var (
	// MySQL dialect.
	Dialect dbstore.DBMsgStoreDialect = nxDialect{}
)

func (d nxDialect) InsertMsg(ctx context.Context, q dbstore.Queryer, table string, id string, subject string, data []byte) error {
	sql := fmt.Sprintf(`INSERT INTO %s (id, subject, data) VALUES (?, ?, ?)`, table)
	_, err := q.ExecContext(ctx, sql, id, subject, data)
	return err
}

func (d nxDialect) DeleteMsgs(ctx context.Context, q dbstore.Queryer, table string, ids []string) error {
	phs := []byte{}
	args := []interface{}{}
	for i, id := range ids {
		if i != 0 {
			phs = append(phs, ", "...)
		}
		phs = append(phs, '?')
		args = append(args, id)
	}
	sql := fmt.Sprintf(`DELETE FROM %s WHERE id IN (%s)`, table, phs)
	_, err := q.ExecContext(ctx, sql, args...)
	return err
}

func (d nxDialect) CreateMsgStoreTable(ctx context.Context, q dbstore.Queryer, table string) error {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id char(20) NOT NULL,
		ts timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
		subject varchar(128) DEFAULT NULL,
		data blob,
		PRIMARY KEY (id),
		KEY ts (ts)
	)`, table)
	_, err := q.ExecContext(ctx, sql)
	return err
}

func (d nxDialect) GetLock(ctx context.Context, conn *sql.Conn, table string) (acquired bool, err error) {
	// Get lock no wait.
	row := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, 0)`, d.lockName(table))
	r := sql.NullInt64{}
	if err := row.Scan(&r); err != nil {
		return false, err
	}
	/*
		Returns 1 if the lock was obtained successfully,
		0 if the attempt timed out (for example, because another client has previously locked the name),
		or NULL if an error occurred (such as running out of memory or the thread was killed with mysqladmin kill).
	*/
	return r.Int64 == 1, nil
}

func (d nxDialect) ReleaseLock(ctx context.Context, conn *sql.Conn, table string) error {
	row := conn.QueryRowContext(ctx, `SELECT RELEASE_LOCK(?)`, d.lockName(table))
	r := sql.NullInt64{}
	return row.Scan(&r)
}

func (d nxDialect) SelectMsgs(ctx context.Context, conn *sql.Conn, table string, window time.Duration) (
	iter func(next bool) (id, subject string, data []byte, err error),
	err error,
) {
	sql := fmt.Sprintf(`SELECT id, subject, data FROM %s WHERE ts <= NOW() - INTERVAL ? SECOND`, table)
	rows, err := conn.QueryContext(ctx, sql, int(window.Seconds()))
	if err != nil {
		return nil, err
	}
	return func(next bool) (id, subject string, data []byte, err error) {
		if next == false {
			rows.Close()
			return
		}
		if rows.Next() {
			err = rows.Scan(&id, &subject, &data)
			if err == nil {
				return
			}
			return "", "", nil, err
		} else {
			return "", "", nil, rows.Err()
		}
	}, nil
}

func (d nxDialect) lockName(table string) string {
	return "npmsg.dbstore.lock:" + table
}
