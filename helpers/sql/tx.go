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

type txContext struct {
	// stacks of functions
	onCommitted []func()
	onFinalised []func()
}

func curTxContext(ctx context.Context) *txContext {
	v := ctx.Value(txContextKey{})
	ret, ok := v.(*txContext)
	if !ok {
		panic(errors.New("Tx context not found, make sure you're calling this inside WithTx"))
	}
	return ret
}

// OnCommitted adds a function which will be only called after the transaction has been successful committed.
// The invocation order of OnCommitted functions just like defer functions.
func OnCommitted(ctx context.Context, fn func()) {
	txCtx := curTxContext(ctx)
	txCtx.onCommitted = append(txCtx.onCommitted, fn)
}

// OnFinalised adds a function which will be called after the transaction ended (either committed or rollback).
// The invocation order of OnCommitted functions just like defer functions.
func OnFinalised(ctx context.Context, fn func()) {
	txCtx := curTxContext(ctx)
	txCtx.onFinalised = append(txCtx.onFinalised, fn)
}

// WithTx starts a transaction and run fn. If no error is returned, the transaction is committed.
// Otherwise it is rollbacked and the error is returned to the caller (except returning Rollback,
// which will rollback the transaction but not return error).
//
// Inside one can use OnCommitted/OnFinalised to add functions to be called after tx end.
func WithTx(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) (err error) {
	txCtx := &txContext{}
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
