package binlogmsg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/huangjunwen/golibs/sqlh"
	"github.com/pkg/errors"
)

// CreateMsgTable creates a msg table to store msgs.
func CreateMsgTable(ctx context.Context, q sqlh.Queryer, schema, table string) error {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s.%s (
		id BIGINT UNSIGNED AUTO_INCREMENT,
		ts TIMESTAMP DEFAULT NOW(),
		subject VARCHAR(128) NOT NULL,
		data BLOB,
		PRIMARY KEY (id))
	`, schema, table)
	_, err := q.ExecContext(ctx, sql)
	return errors.WithMessagef(err, "Create msg table %s.%s error", schema, table)
}

// List all msg table names. NOTE: since 8.0, information.tables is also innodb.
func listMsgTables(ctx context.Context, q sqlh.Queryer, filter MsgTableFilter) (schemas, tables []string, err error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			table_schema, table_name
		FROM information_schema.tables
		WHERE
			table_type = 'BASE TABLE' AND
			table_schema NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
	`)
	if err != nil {
		err = errors.WithMessage(err, "List msg tables error")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		if err = rows.Scan(&schema, &table); err != nil {
			err = errors.WithMessage(err, "List msg tables scan error")
			return
		}
		if !filter(schema, table) {
			continue
		}
		schemas = append(schemas, schema)
		tables = append(tables, table)
	}

	return
}

func getLock(ctx context.Context, q sqlh.Queryer, lockName string) (ok bool, err error) {
	r := sql.NullInt64{}
	if err := q.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", lockName).Scan(&r); err != nil {
		return false, errors.WithMessage(err, "Get lock error")
	}
	return r.Int64 == 1, nil
}

func releaseLock(ctx context.Context, q sqlh.Queryer, lockName string) error {
	r := sql.NullInt64{}
	return errors.WithMessage(q.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", lockName).Scan(&r), "Release lock error")
}

func addMsg(ctx context.Context, q sqlh.Queryer, schema, table, subject string, data []byte) error {
	_, err := q.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s.%s (subject, data) VALUES (?, ?)
	`, schema, table), subject, data)
	return errors.WithMessage(err, "Insert message error")
}

func delMsg(ctx context.Context, q sqlh.Queryer, schema, table string, id uint64) error {
	_, err := q.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM %s.%s WHERE id=?
	`, schema, table), id)
	return errors.WithMessage(err, "Delete message error")
}
