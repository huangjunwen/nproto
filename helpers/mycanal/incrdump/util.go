package incrdump

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
	. "github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
)

func safeUint64Minus(left, right uint64) uint64 {
	if left >= right {
		return left - right
	}
	panic(fmt.Errorf("%d < %d", left, right))
}

func gtidFromGTIDEvent(e *replication.GTIDEvent) string {
	return fmt.Sprintf(
		"%s:%d",
		uuid.Must(uuid.FromBytes(e.SID)).String(),
		e.GNO,
	)
}

// convert bytes to string.
func handleBytes(data []interface{}) {
	for i, v := range data {
		if b, ok := v.([]byte); ok {
			data[i] = string(b)
		}
	}
}

// NOTE: go-mysql stores int as signed values since before MySQL-8, no signedness
// information is presents in binlog.
func handleUnsigned(data []interface{}, e *replication.TableMapEvent, unsignedMap map[int]bool) {
	for i, typ := range e.ColumnType {
		if !isNumericColumn(e, i) {
			continue
		}

		if !unsignedMap[i] {
			continue
		}

		// Copy from go-mysql/canal/rows.go
		switch v := data[i].(type) {
		case int8:
			data[i] = uint8(v)

		case int16:
			data[i] = uint16(v)

		case int32:
			if v < 0 && typ == MYSQL_TYPE_INT24 {
				// 16777215 is the maximum value of mediumint
				data[i] = uint32(16777215 + v + 1)
			} else {
				data[i] = uint32(v)
			}

		case int64:
			data[i] = uint64(v)

		case int:
			data[i] = uint(v)

		default:
			// float/double ...

		}
	}
}

/*
	TODO:
	My PR has not merged yet: https://github.com/siddontang/go-mysql/pull/482
	So copy here.
*/

func unsignedMap(e *replication.TableMapEvent) map[int]bool {
	if len(e.SignednessBitmap) == 0 {
		return nil
	}
	p := 0
	ret := make(map[int]bool)
	for i := 0; i < int(e.ColumnCount); i++ {
		if !isNumericColumn(e, i) {
			continue
		}
		ret[i] = e.SignednessBitmap[p/8]&(1<<uint(7-p%8)) != 0
		p++
	}
	return ret
}

func isNumericColumn(e *replication.TableMapEvent, i int) bool {
	switch realType(e, i) {
	case MYSQL_TYPE_TINY,
		MYSQL_TYPE_SHORT,
		MYSQL_TYPE_INT24,
		MYSQL_TYPE_LONG,
		MYSQL_TYPE_LONGLONG,
		MYSQL_TYPE_NEWDECIMAL,
		MYSQL_TYPE_FLOAT,
		MYSQL_TYPE_DOUBLE:
		return true

	default:
		return false
	}
}

func realType(e *replication.TableMapEvent, i int) byte {
	typ := e.ColumnType[i]
	meta := e.ColumnMeta[i]

	switch typ {
	case MYSQL_TYPE_STRING:
		rtyp := byte(meta >> 8)
		if rtyp == MYSQL_TYPE_ENUM || rtyp == MYSQL_TYPE_SET {
			return rtyp
		}

	case MYSQL_TYPE_DATE:
		return MYSQL_TYPE_NEWDATE
	}

	return typ
}
