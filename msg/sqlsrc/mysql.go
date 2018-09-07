package sqlsrc

import (
	"database/sql"
	"fmt"
)

type mysqlDialect struct {
	tableName  string
	createStmt string
	insertStmt string
	selectStmt string
}

func newMySQLDialect(tableName string) sqlMsgSourceDialect {
	return &mysqlDialect{
		tableName: tableName,
		createStmt: `
			CREATE TABLE IF NOT EXISTS ` + "`" + tableName + "`" + ` (
				id INT NOT NULL AUTO_INCREMENT,
				subject VARCHAR(128) NOT NULL DEFAULT "",
				data BLOB,
				PRIMARY KEY (id)
			)
		`,
		insertStmt: fmt.Sprintf("INSERT INTO `%s` (subject, data) VALUES (?, ?)", tableName),
		selectStmt: fmt.Sprintf("SELECT id, subject, data FROM `%s` ORDER BY id", tableName),
	}
}

func (dialect *mysqlDialect) CreateStmt() string {
	return dialect.createStmt
}

func (dialect *mysqlDialect) InsertStmt() string {
	return dialect.insertStmt
}

func (dialect *mysqlDialect) SelectStmt() string {
	return dialect.selectStmt
}

func (dialect *mysqlDialect) DeleteStmt(n int) string {
	phs := []byte{}
	for i := 0; i < n; i++ {
		if i != 0 {
			phs = append(phs, ", "...)
		}
		phs = append(phs, '?')
	}

	return fmt.Sprintf("DELETE FROM `%s` WHERE id IN (%s)", dialect.tableName, phs)
}

// NewMySQLMsgSource creates a SQLMsgSource backed by MySQL. `tableName` is the mysql table to store messages.
// It will be created if not exists.
func NewMySQLMsgSource(db *sql.DB, tableName string, opts ...SQLMsgSourceOption) (*SQLMsgSource, error) {
	return newSQLMsgSource(newMySQLDialect, db, tableName, opts...)
}
