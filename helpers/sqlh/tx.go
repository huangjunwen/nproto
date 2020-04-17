package sqlh

import (
	"context"
	"database/sql"
	"errors"
)

var (
	// Rollback is used to rollback a transaction without returning an error.
	Rollback = errors.New("Just rollback")
)

type txContextKey struct{}

// TxContext contains transaction context.
type TxContext struct {
	db *sql.DB
	// tx associated local variables
	locals map[interface{}]interface{}
	// stacks of functions
	onCommitted []func()
	onFinalised []func()
}

// CurTxContext returns current TxContext in transaction.
func CurTxContext(ctx context.Context) *TxContext {
	v := ctx.Value(txContextKey{})
	ret, _ := v.(*TxContext)
	return ret
}

// MustCurTxContext is the `must` version of CurTxContext.
func MustCurTxContext(ctx context.Context) *TxContext {
	ret := CurTxContext(ctx)
	if ret == nil {
		panic(errors.New("MustCurTxContext must be called within WithTx"))
	}
	return ret
}

// DB returns the original *sql.DB that starts the transaction.
func (txCtx *TxContext) DB() *sql.DB {
	return txCtx.db
}

// SetLocal sets tx local variable.
func (txCtx *TxContext) SetLocal(key, val interface{}) {
	txCtx.locals[key] = val
}

// Local gets tx local variable.
func (txCtx *TxContext) Local(key interface{}) interface{} {
	return txCtx.locals[key]
}

// OnCommitted adds a function which will be only called after the transaction has been successful committed.
// The invocation order of OnCommitted functions just like defer functions.
func (txCtx *TxContext) OnCommitted(fn func()) {
	txCtx.onCommitted = append(txCtx.onCommitted, fn)
}

// OnFinalised adds a function which will be called after the transaction ended (either committed or rollback).
// The invocation order of OnFinalised functions just like defer functions.
func (txCtx *TxContext) OnFinalised(fn func()) {
	txCtx.onFinalised = append(txCtx.onFinalised, fn)
}

// WithTx starts a transaction and run fn. If no error is returned, the transaction is committed.
// Otherwise it is rollbacked and the error is returned to the caller (except returning Rollback,
// which will rollback the transaction but not return error).
//
// Inside one can use OnCommitted/OnFinalised to add functions to be called after tx end.
func WithTx(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) (err error) {
	txCtx := &TxContext{
		db:     db,
		locals: make(map[interface{}]interface{}),
	}
	ctx = context.WithValue(ctx, txContextKey{}, txCtx)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return
	}

	defer func() {
		tx.Rollback()
		for i := len(txCtx.onFinalised) - 1; i >= 0; i-- {
			txCtx.onFinalised[i]()
		}
		if err == Rollback {
			// Rollback is not treated as an error.
			err = nil
		}
	}()

	err = fn(ctx, tx)
	if err != nil {
		return
	}

	err = tx.Commit()
	if err != nil {
		return
	}

	for i := len(txCtx.onCommitted) - 1; i >= 0; i-- {
		txCtx.onCommitted[i]()
	}

	return
}
