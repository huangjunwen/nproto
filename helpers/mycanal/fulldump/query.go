package fulldump

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/huangjunwen/nproto/helpers/sqlh"
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

	names, err := rows.Columns()
	if err != nil {
		return nil, errors.WithMessage(err, "fulldump.Query get Columns error")
	}

	makeScanValues := makeScanValues(rows)
	values := []interface{}(nil)

	return func(next bool) (map[string]interface{}, error) {
		if !next {
			return nil, errors.WithMessage(rows.Close(), "fulldump.Query close rows error")
		}

		if !rows.Next() {
			return nil, errors.WithMessage(rows.Err(), "fulldump.Query rows error")
		}

		values = makeScanValues(values)
		if err := rows.Scan(values...); err != nil {
			return nil, errors.WithMessage(err, "fulldump.Query scan error")
		}
		postProcessScanedValues(values)

		m := make(map[string]interface{})
		for i, name := range names {
			m[name] = values[i]
		}

		return m, nil
	}, nil
}

// FullTableQuery full dump a table.
func FullTableQuery(ctx context.Context, q sqlh.Queryer, dbName, table string) (RowIter, error) {
	query := fmt.Sprintf("SELECT * FROM %s.%s", dbName, table)
	return Query(ctx, q, query)
}
