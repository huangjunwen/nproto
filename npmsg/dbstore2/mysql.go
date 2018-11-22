package dbstore

import (
	"context"
	"fmt"
	"strconv"
)

type mysqlDialect struct{}

func (d mysqlDialect) CreateSQL(table string) string {
	return fmt.Sprintf(`
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
}

func (d mysqlDialect) InsertMsg(ctx context.Context, q Queryer, table string, batch string, subject string, data []byte) (id int64, err error) {
	sql := fmt.Sprintf("INSERT INTO %s (batch, subject, data) VALUES (?, ?, ?)", table)
	r, err := q.ExecContext(ctx, sql, batch, subject, data)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func (d mysqlDialect) DeleteMsgs(ctx context.Context, q Queryer, table string, ids []int64) error {
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
