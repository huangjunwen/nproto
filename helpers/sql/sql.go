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

// TxPlugin is used to add extra functions during transaction life time.
type TxPlugin interface {
	// TxInitialized will be called after a transaction started. If an error returned, the transaction
	// will be rollbacked at once.
	TxInitialized(db *sql.DB, tx Queryer) error
	// TxCommitted will be called only after the transaction has been successfully committed.
	TxCommitted()
	// TxFinalised will be called after the transaction committed/rollbacked.
	TxFinalised()
}

// BaseTxPlugin do nothing.
type BaseTxPlugin struct{}

var (
	_ Queryer  = (*sql.DB)(nil)
	_ Queryer  = (*sql.Conn)(nil)
	_ Queryer  = (*sql.Tx)(nil)
	_ TxPlugin = BaseTxPlugin{}
)

// TxInitialized implements TxPlugin interface.
func (p BaseTxPlugin) TxInitialized(db *sql.DB, tx Queryer) error {
	return nil
}

// TxCommitted implements TxPlugin interface.
func (p BaseTxPlugin) TxCommitted() {
	return
}

// TxFinalised implements TxPlugin interface.
func (p BaseTxPlugin) TxFinalised() {
	return
}

// WithTx starts a transaction and run fn. If no error is returned, the transaction is committed.
// Otherwise it is rollbacked.
func WithTx(db *sql.DB, fn func(Queryer) error, plugins ...TxPlugin) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var n int
	defer func() {
		tx.Rollback()
		for _, plugin := range plugins[:n] {
			plugin.TxFinalised()
		}
	}()

	for _, plugin := range plugins {
		if err := plugin.TxInitialized(db, tx); err != nil {
			return err
		}
		n += 1
	}

	err = fn(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	for _, plugin := range plugins {
		plugin.TxCommitted()
	}
	return nil
}
