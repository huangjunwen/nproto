package fulldump

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	sqlh "github.com/huangjunwen/nproto/helpers/sql"
)

// RowIter is used for result set iteration. It returns nil if no more row.
// Caller should invoke RowIter(false) to close the iterator and release resource.
type RowIter func(next bool) (map[string]interface{}, error)

// Query and returns RowIter
func Query(ctx context.Context, q sqlh.Queryer, query string, args ...interface{}) (iter RowIter, err error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.WithMessage(err, "fulldump.Query error")
	}
	defer func() {
		if err != nil {
			rows.Close()
		}
	}()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, errors.WithMessage(err, "fulldump.Query get column types error")
	}

	return func(next bool) (map[string]interface{}, error) {
		if !next {
			return nil, errors.WithMessage(rows.Close(), "fulldump.Query close rows error")
		}

		if !rows.Next() {
			return nil, errors.WithMessage(rows.Err(), "fulldump.Query rows error")
		}

		pointers := []reflect.Value{}
		values := []interface{}{}
		for _, colType := range colTypes {
			pointer := reflect.New(colType.ScanType())
			pointers = append(pointers, pointer)
			values = append(values, pointer.Interface())
		}

		if err := rows.Scan(values...); err != nil {
			return nil, errors.WithMessage(err, "fulldump.Query scan error")
		}

		m := make(map[string]interface{})

		for i, colType := range colTypes {

			var val interface{}
			// see https://github.com/go-sql-driver/mysql/blob/master/fields.go#L101
			switch v := pointers[i].Elem().Interface().(type) {
			case sql.NullBool:
				if v.Valid {
					val = v.Bool
				}

			case sql.NullFloat64:
				if v.Valid {
					val = v.Float64
				}

			case sql.NullInt32:
				if v.Valid {
					val = v.Int32
				}

			case sql.NullInt64:
				if v.Valid {
					val = v.Int64
				}

			case sql.NullString:
				if v.Valid {
					val = v.String
				}

			case sql.NullTime:
				if v.Valid {
					val = v.Time.UTC()
				}

			case sql.RawBytes:
				if len(v) != 0 {
					val = string(v)
				}

			case time.Time:
				val = v.UTC()

			case mysql.NullTime:
				if v.Valid {
					val = v.Time.UTC()
				}

			default:
				val = v
			}

			m[colType.Name()] = val
		}

		return m, nil
	}, nil
}

// FullTableQuery full dump a table.
func FullTableQuery(ctx context.Context, q sqlh.Queryer, dbName, table string) (RowIter, error) {
	query := fmt.Sprintf("SELECT * FROM %s.%s", dbName, table)
	return Query(ctx, q, query)
}
