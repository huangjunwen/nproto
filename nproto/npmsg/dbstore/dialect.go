package dbstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	sqlh "github.com/huangjunwen/nproto/helpers/sql"
)

type mysqlDialect struct{}

var (
	_ dbStoreDialect = mysqlDialect{}
)

func (d mysqlDialect) CreateTable(ctx context.Context, q sqlh.Queryer, table string) error {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id BIGINT UNSIGNED AUTO_INCREMENT,
		ts TIMESTAMP DEFAULT NOW(),
		batch VARCHAR(32) NOT NULL,
		subject VARCHAR(128) NOT NULL,
		data BLOB,
		KEY (batch),
		KEY (ts),
		PRIMARY KEY (id)
	)`, table)
	_, err := q.ExecContext(ctx, sql)
	return err
}

func (d mysqlDialect) InsertMsg(ctx context.Context, q sqlh.Queryer, table string, batch string, subject string, data []byte) (id int64, err error) {
	sql := fmt.Sprintf("INSERT INTO %s (batch, subject, data) VALUES (?, ?, ?)", table)
	r, err := q.ExecContext(ctx, sql, batch, subject, data)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d mysqlDialect) DeleteMsgs(ctx context.Context, q sqlh.Queryer, table string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	sql := make([]byte, 0, 512)
	sql = append(sql, "DELETE FROM "...)
	sql = append(sql, table...)
	sql = append(sql, " WHERE id IN ("...)
	for i, id := range ids {
		if i != 0 {
			sql = append(sql, ", "...)
		}
		sql = strconv.AppendInt(sql, id, 10)
	}
	sql = append(sql, ')')
	_, err := q.ExecContext(ctx, string(sql))
	return err
}

func (d mysqlDialect) SelectMsgsByBatch(ctx context.Context, q sqlh.Queryer, table, batch string) msgStream {
	sql := fmt.Sprintf("SELECT id, subject, data FROM %s WHERE batch=?", table)
	rows, err := q.QueryContext(ctx, sql, batch)
	if err != nil {
		return newErrStream(err)
	}
	return func(next bool) (*msgNode, error) {
		if !next {
			return nil, rows.Close()
		}
		if !rows.Next() {
			return nil, rows.Err()
		}
		node := &msgNode{}
		err := rows.Scan(&node.Id, &node.Subject, &node.Data)
		if err != nil {
			return nil, err
		}
		return node, nil
	}
}

func (d mysqlDialect) SelectMsgsAll(ctx context.Context, q sqlh.Queryer, table string, tsDelta time.Duration) msgStream {
	sql := fmt.Sprintf("SELECT id, subject, data FROM %s WHERE ts <= NOW() - INTERVAL ? SECOND", table)
	rows, err := q.QueryContext(ctx, sql, uint64(tsDelta.Seconds()))
	if err != nil {
		return newErrStream(err)
	}
	return func(next bool) (*msgNode, error) {
		if !next {
			return nil, rows.Close()
		}
		if !rows.Next() {
			return nil, rows.Err()
		}
		node := &msgNode{}
		err := rows.Scan(&node.Id, &node.Subject, &node.Data)
		if err != nil {
			return nil, err
		}
		return node, nil
	}
}

func (d mysqlDialect) GetLock(ctx context.Context, conn *sql.Conn, table string) (acquired bool, err error) {
	// Get lock no wait.
	r := sql.NullInt64{}
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", d.lockName(table)).Scan(&r); err != nil {
		return false, err
	}
	/*
		Returns 1 if the lock was obtained successfully,
		0 if the attempt timed out (for example, because another client has previously locked the name),
		or NULL if an error occurred (such as running out of memory or the thread was killed with mysqladmin kill).
	*/
	return r.Int64 == 1, nil
}

func (d mysqlDialect) ReleaseLock(ctx context.Context, conn *sql.Conn, table string) error {
	r := sql.NullInt64{}
	return conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", d.lockName(table)).Scan(&r)
}

func (d mysqlDialect) lockName(table string) string {
	return "npmsg.dbstore.lock:" + table
}
