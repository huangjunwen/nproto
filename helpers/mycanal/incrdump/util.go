package incrdump

import (
	"fmt"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
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

func normalizeRowData(
	data []interface{},
	meta *tableMeta,
) {
	for i, val := range data {

		// No need to handle nil.
		if val == nil {
			continue
		}

		// NOTE: go-mysql stores int as signed values since before MySQL-8, no signedness
		// information is presents in binlog. So we need to convert here if it is unsigned.
		if isNumericColumn(meta.Table, i) {
			if v, ok := val.(decimal.Decimal); ok {
				data[i] = v.String()
				continue
			}

			if !meta.UnsignedMap[i] {
				continue
			}

			typ := realType(meta.Table, i)
			// Copy from go-mysql/canal/rows.go
			switch v := val.(type) {
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
			continue
		}

		if isEnumColumn(meta.Table, i) {
			v, ok := val.(int64)
			if !ok {
				panic(fmt.Errorf("Expect int64 for enum (MYSQL_TYPE_ENUM) field but got %T %#v", val, val))
			}
			data[i] = meta.EnumStrValueMap[i][int(v)-1]
			continue
		}

		if isSetColumn(meta.Table, i) {
			v, ok := val.(int64)
			if !ok {
				panic(fmt.Errorf("Expect int64 for set (MYSQL_TYPE_SET) field but got %T %#v", val, val))
			}
			setStrValue := meta.SetStrValueMap[i]
			vals := []string{}
			for j := 0; j < 64; j++ {
				if (v & (1 << uint(j))) != 0 {
					vals = append(vals, setStrValue[j])
				}
			}
			data[i] = strings.Join(vals, ",")
			continue
		}

		if realType(meta.Table, i) == MYSQL_TYPE_YEAR {
			v, ok := val.(int)
			if !ok {
				panic(fmt.Errorf("Expect int for year (MYSQL_TYPE_YEAR) field but got %T %#v", val, val))
			}
			// NOTE: Convert to uint16 to keep the same as fulldump.
			data[i] = uint16(v)
			continue
		}

		if realType(meta.Table, i) == MYSQL_TYPE_NEWDATE {
			v, ok := val.(string)
			if !ok {
				panic(fmt.Errorf("Expect string for date (MYSQL_TYPE_NEWDATE) field but got %T %#v", val, val))
			}
			// NOTE: Convert to time.Time to keep the same as fulldump.
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				panic(err)
			}
			data[i] = t
			continue
		}

		switch v := val.(type) {
		case time.Time:
			data[i] = v.UTC()

		case []byte:
			data[i] = string(v)
		}
	}
}

/*
	TODO:
	My PR has not merged yet: https://github.com/siddontang/go-mysql/pull/482
	So copy here.
*/

type tableMeta struct {
	Table           *replication.TableMapEvent
	UnsignedMap     map[int]bool
	EnumStrValueMap map[int][]string
	SetStrValueMap  map[int][]string
}

func newTableMeta(e *replication.TableMapEvent) *tableMeta {
	return &tableMeta{
		Table:           e,
		UnsignedMap:     unsignedMap(e),
		EnumStrValueMap: enumStrValueMap(e),
		SetStrValueMap:  setStrValueMap(e),
	}
}

func enumStrValueMap(e *replication.TableMapEvent) map[int][]string {
	return strValueMap(e, isEnumColumn, e.EnumStrValueString())
}

func setStrValueMap(e *replication.TableMapEvent) map[int][]string {
	return strValueMap(e, isSetColumn, e.SetStrValueString())
}

func strValueMap(
	e *replication.TableMapEvent,
	includeType func(*replication.TableMapEvent, int) bool,
	strValue [][]string,
) map[int][]string {

	if len(strValue) == 0 {
		return nil
	}
	p := 0
	ret := make(map[int][]string)
	for i := 0; i < int(e.ColumnCount); i++ {
		if !includeType(e, i) {
			continue
		}
		ret[i] = strValue[p]
		p++
	}
	return ret
}

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

func isEnumColumn(e *replication.TableMapEvent, i int) bool {
	return realType(e, i) == MYSQL_TYPE_ENUM
}

func isSetColumn(e *replication.TableMapEvent, i int) bool {
	return realType(e, i) == MYSQL_TYPE_SET
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
