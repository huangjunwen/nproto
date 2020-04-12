package incrdump

import (
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
)

// TrxContext is the context of binlog trx.
type TrxContext struct {
	prevGset  mysql.GTIDSet
	gtidEvent *replication.GTIDEvent

	// cache fields
	gtid      string
	afterGset mysql.GTIDSet
}

// GTID returns the gtid for current trx.
func (trxCtx *TrxContext) GTID() string {
	if trxCtx.gtid == "" {
		trxCtx.gtid = gtidFromGTIDEvent(trxCtx.gtidEvent)
	}
	return trxCtx.gtid
}

// PrevGTIDSet returns the gtid set before current trx.
func (trxCtx *TrxContext) PrevGTIDSet() mysql.GTIDSet {
	return trxCtx.prevGset
}

// AfterGTIDSet returns the gtid set after current trx.
func (trxCtx *TrxContext) AfterGTIDSet() mysql.GTIDSet {
	if trxCtx.afterGset == nil {
		afterGset := trxCtx.prevGset.Clone()
		if err := afterGset.Update(trxCtx.GTID()); err != nil {
			panic(err)
		}
		trxCtx.afterGset = afterGset
	}
	return trxCtx.afterGset
}
