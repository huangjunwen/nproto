package incrdump

import (
	"context"

	"github.com/siddontang/go-mysql/replication"
)

// Handler is used to handle events in binlog, can be one of the followings:
//
//   - TrxBeginning: the beginning of a gtid trx
//   - TrxEnding: the end of a gtid trx
//   - RowInsertion: row insert, between TrxBeginning/TrxEnding
//   - RowUpdating: row update, between TrxBeginning/TrxEnding
//   - RowDeletion: row delete, between TrxBeginning/TrxEnding
//
// Maybe more events will be added in the future
type Handler func(ctx context.Context, e interface{}) error

// TrxBeginning represents the start of a trx.
type TrxBeginning TrxContext

// TrxEnding represents the end of a trx.
type TrxEnding TrxContext

// TrxEvent represents event inside a trx.
type TrxEvent interface {
	// TrxContext returns the trx context.
	TrxContext() *TrxContext
}

// RowChange is the common interface of RowInsertion/RowUpdating/RowDeletion.
type RowChange interface {
	TrxEvent

	// SchemaName returns the database name.
	SchemaName() string

	// TableName returns the table name.
	TableName() string

	// ColumnNames returns column names of the table.
	ColumnNames() []string

	// BeforeData returns column data before the change or nil if not applicable.
	BeforeData() []interface{}

	// BeforeDataMap returns data map (column name -> column data)
	// before the change or nil if not applicable.
	BeforeDataMap() map[string]interface{}

	// AfterData returns column data after the change or nil if not applicable.
	AfterData() []interface{}

	// AfterDataMap returns data map (column name -> column data)
	// after the change or nil if not applicable.
	AfterDataMap() map[string]interface{}
}

// RowInsertion represents a row insertion.
type RowInsertion struct {
	*rowChange
}

// RowUpdating represents a row updating.
type RowUpdating struct {
	*rowChange
}

// RowDeletion represents a row deletion.
type RowDeletion struct {
	*rowChange
}

type rowChange struct {
	trxCtx    *TrxContext
	rowsEvent *replication.RowsEvent

	schemaName string
	tableName  string
	beforeData []interface{}
	afterData  []interface{}
}

var (
	_ TrxEvent  = (*TrxBeginning)(nil)
	_ TrxEvent  = (*TrxEnding)(nil)
	_ RowChange = (*RowInsertion)(nil)
	_ RowChange = (*RowUpdating)(nil)
	_ RowChange = (*RowDeletion)(nil)
)

// TrxContext returns the trx context.
func (e *TrxBeginning) TrxContext() *TrxContext {
	return (*TrxContext)(e)
}

// TrxContext returns the trx context.
func (e *TrxEnding) TrxContext() *TrxContext {
	return (*TrxContext)(e)
}

// TrxContext returns the trx context.
func (e *rowChange) TrxContext() *TrxContext {
	return e.trxCtx
}

// SchemaName returns the database name.
func (e *rowChange) SchemaName() string {
	return e.schemaName
}

// TableName returns the table name.
func (e *rowChange) TableName() string {
	return e.tableName
}

// ColumnNames returns column names of the table.
func (e *rowChange) ColumnNames() []string {
	return e.rowsEvent.Table.ColumnNameString()
}

// BeforeData returns column data before the change or nil if not applicable.
func (e *rowChange) BeforeData() []interface{} {
	return e.beforeData
}

// BeforeDataMap returns data map (column name -> column data)
// before the change or nil if not applicable.
func (e *rowChange) BeforeDataMap() map[string]interface{} {
	if e.beforeData == nil {
		return nil
	}
	ret := make(map[string]interface{})
	for i, name := range e.ColumnNames() {
		ret[name] = e.beforeData[i]
	}
	return ret
}

// AfterData returns column data after the change or nil if not applicable.
func (e *rowChange) AfterData() []interface{} {
	return e.afterData
}

// AfterDataMap returns data map (column name -> column data)
// after the change or nil if not applicable.
func (e *rowChange) AfterDataMap() map[string]interface{} {
	if e.afterData == nil {
		return nil
	}
	ret := make(map[string]interface{})
	for i, name := range e.ColumnNames() {
		ret[name] = e.afterData[i]
	}
	return ret
}
