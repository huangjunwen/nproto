package fulldump

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/go-sql-driver/mysql"
	"gopkg.in/volatiletech/null.v6"
)

func makeScanValues(rows *sql.Rows) func([]interface{}) []interface{} {

	// XXX: Using ColumnType.ScanType can't handle extreme large value for BIGINT UNSIGNED DEFAULT NULL
	// columns because 'github.com/go-sql-driver/mysql' uses sql.NullInt64.
	// And there is no way to know unsigned flag from public api of 'database/sql'.
	// So we have to query the private fields here.

	intlCols := reflect.
		ValueOf(rows).         // *sql.Rows
		Elem().                // sql.Rows
		FieldByName("rowsi").  // driver.Rows
		Elem().                // *mysql.textRows or *mysql.binaryRows
		Elem().                // mysql.textRows or mysql.binaryRows
		FieldByName("rs").     // mysql.resultSet
		FieldByName("columns") // []mysql.mysqlField

	n := intlCols.Len()
	fns := make([]func() interface{}, 0, n)

	for i := 0; i < n; i++ {

		intlCol := intlCols.Index(i)
		fieldType := fieldType(intlCol.FieldByName("fieldType").Uint())
		flags := fieldFlag(intlCol.FieldByName("flags").Uint())

		// The following are copied and modified from github.com/go-sql-driver/mysql@v1.5.0/fields.go mysqlField.scanType

		notNull := (flags & flagNotNULL) != 0
		unsigned := (flags & flagUnsigned) != 0

		switch fieldType {
		case fieldTypeTiny:
			if notNull {
				if unsigned {
					fns = append(fns, newUint8)
				} else {
					fns = append(fns, newInt8)
				}
			} else {
				if unsigned {
					fns = append(fns, newNullUint8)
				} else {
					fns = append(fns, newNullInt8)
				}
			}

		case fieldTypeShort, fieldTypeYear:
			if notNull {
				if unsigned {
					fns = append(fns, newUint16)
				} else {
					fns = append(fns, newInt16)
				}
			} else {
				if unsigned {
					fns = append(fns, newNullUint16)
				} else {
					fns = append(fns, newNullInt16)
				}
			}

		case fieldTypeInt24, fieldTypeLong:
			if notNull {
				if unsigned {
					fns = append(fns, newUint32)
				} else {
					fns = append(fns, newInt32)
				}
			} else {
				if unsigned {
					fns = append(fns, newNullUint32)
				} else {
					fns = append(fns, newNullInt32)
				}
			}

		case fieldTypeLongLong:
			if notNull {
				if unsigned {
					fns = append(fns, newUint64)
				} else {
					fns = append(fns, newInt64)
				}
			} else {
				if unsigned {
					fns = append(fns, newNullUint64)
				} else {
					fns = append(fns, newNullInt64)
				}
			}

		case fieldTypeFloat:
			if notNull {
				fns = append(fns, newFloat32)
			} else {
				fns = append(fns, newNullFloat32)
			}

		case fieldTypeDouble:
			if notNull {
				fns = append(fns, newFloat64)
			} else {
				fns = append(fns, newNullFloat64)
			}

		case fieldTypeDecimal, fieldTypeNewDecimal, fieldTypeVarChar,
			fieldTypeBit, fieldTypeEnum, fieldTypeSet, fieldTypeTinyBLOB,
			fieldTypeMediumBLOB, fieldTypeLongBLOB, fieldTypeBLOB,
			fieldTypeVarString, fieldTypeString, fieldTypeGeometry, fieldTypeJSON,
			fieldTypeTime:
			fns = append(fns, newNullString)

		case fieldTypeDate, fieldTypeNewDate,
			fieldTypeTimestamp, fieldTypeDateTime:
			fns = append(fns, newNullTime)

		default:
			panic(fmt.Errorf("Don't known field type %d", fieldType))

		}
	}

	return func(slice []interface{}) []interface{} {
		slice = slice[:0]
		for _, fn := range fns {
			slice = append(slice, fn())
		}
		return slice
	}
}

func postProcessScanedValues(vals []interface{}) {

	for i, val := range vals {
		switch v := val.(type) {
		case *int8:
			vals[i] = *v

		case *uint8:
			vals[i] = *v

		case *null.Int8:
			if v.Valid {
				vals[i] = v.Int8
			} else {
				vals[i] = nil
			}

		case *null.Uint8:
			if v.Valid {
				vals[i] = v.Uint8
			} else {
				vals[i] = nil
			}

		case *int16:
			vals[i] = *v

		case *uint16:
			vals[i] = *v

		case *null.Int16:
			if v.Valid {
				vals[i] = v.Int16
			} else {
				vals[i] = nil
			}

		case *null.Uint16:
			if v.Valid {
				vals[i] = v.Uint16
			} else {
				vals[i] = nil
			}

		case *int32:
			vals[i] = *v

		case *uint32:
			vals[i] = *v

		case *null.Int32:
			if v.Valid {
				vals[i] = v.Int32
			} else {
				vals[i] = nil
			}

		case *null.Uint32:
			if v.Valid {
				vals[i] = v.Uint32
			} else {
				vals[i] = nil
			}

		case *int64:
			vals[i] = *v

		case *uint64:
			vals[i] = *v

		case *null.Int64:
			if v.Valid {
				vals[i] = v.Int64
			} else {
				vals[i] = nil
			}

		case *null.Uint64:
			if v.Valid {
				vals[i] = v.Uint64
			} else {
				vals[i] = nil
			}

		case *float32:
			vals[i] = *v

		case *null.Float32:
			if v.Valid {
				vals[i] = v.Float32
			} else {
				vals[i] = nil
			}

		case *float64:
			vals[i] = *v

		case *null.Float64:
			if v.Valid {
				vals[i] = v.Float64
			} else {
				vals[i] = nil
			}

		case *null.String:
			if v.Valid {
				vals[i] = v.String
			} else {
				vals[i] = nil
			}

		case *mysql.NullTime:
			if v.Valid {
				vals[i] = v.Time
			} else {
				vals[i] = nil
			}

		default:
			panic(fmt.Errorf("Unexpected scaned value %#v", v))
		}
	}

}

func newInt8() interface{}        { return new(int8) }
func newUint8() interface{}       { return new(uint8) }
func newNullInt8() interface{}    { return &null.Int8{} }
func newNullUint8() interface{}   { return &null.Uint8{} }
func newInt16() interface{}       { return new(int16) }
func newUint16() interface{}      { return new(uint16) }
func newNullInt16() interface{}   { return &null.Int16{} }
func newNullUint16() interface{}  { return &null.Uint16{} }
func newInt32() interface{}       { return new(int32) }
func newUint32() interface{}      { return new(uint32) }
func newNullInt32() interface{}   { return &null.Int32{} }
func newNullUint32() interface{}  { return &null.Uint32{} }
func newInt64() interface{}       { return new(int64) }
func newUint64() interface{}      { return new(uint64) }
func newNullInt64() interface{}   { return &null.Int64{} }
func newNullUint64() interface{}  { return &null.Uint64{} }
func newFloat32() interface{}     { return new(float32) }
func newNullFloat32() interface{} { return &null.Float32{} }
func newFloat64() interface{}     { return new(float64) }
func newNullFloat64() interface{} { return &null.Float64{} }
func newNullString() interface{}  { return &null.String{} }
func newNullTime() interface{}    { return &mysql.NullTime{} }

// The following are copied from github.com/go-sql-driver/mysql@v1.5.0/const.go

type fieldType byte

const (
	fieldTypeDecimal fieldType = iota
	fieldTypeTiny
	fieldTypeShort
	fieldTypeLong
	fieldTypeFloat
	fieldTypeDouble
	fieldTypeNULL
	fieldTypeTimestamp
	fieldTypeLongLong
	fieldTypeInt24
	fieldTypeDate
	fieldTypeTime
	fieldTypeDateTime
	fieldTypeYear
	fieldTypeNewDate
	fieldTypeVarChar
	fieldTypeBit
)
const (
	fieldTypeJSON fieldType = iota + 0xf5
	fieldTypeNewDecimal
	fieldTypeEnum
	fieldTypeSet
	fieldTypeTinyBLOB
	fieldTypeMediumBLOB
	fieldTypeLongBLOB
	fieldTypeBLOB
	fieldTypeVarString
	fieldTypeString
	fieldTypeGeometry
)

type fieldFlag uint16

const (
	flagNotNULL fieldFlag = 1 << iota
	flagPriKey
	flagUniqueKey
	flagMultipleKey
	flagBLOB
	flagUnsigned
	flagZeroFill
	flagBinary
	flagEnum
	flagAutoIncrement
	flagTimestamp
	flagSet
	flagUnknown1
	flagUnknown2
	flagUnknown3
	flagUnknown4
)
