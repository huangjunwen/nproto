package sqlh

import (
	"context"
	"database/sql"
)

// Queryer abstracts sql.DB/sql.Conn/sql.Tx .
type Queryer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

var (
	_ Queryer = (*sql.DB)(nil)
	_ Queryer = (*sql.Conn)(nil)
	_ Queryer = (*sql.Tx)(nil)
)
