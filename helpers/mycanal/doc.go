// Package mycanal is a CDC (Change Data Capture) library for MySQL8+
//
// It provides helper functions for full dump (package fulldump) and incremental change capture (package incrdump).
// Prerequisites:
//   - MySQL-8.0.2 and above
//   - GTID mode enabled:
//     - `--gtid-mode=ON`
//     - `--enforce-gtid-consistency=ON`
//   - binlog enabled with the following:
//     - `--log-bin=xxxx`: enable bin log
//     - `--server-id=xxx`: the server id
//     - `--binlog-format=ROW`: binlog output row changes instead of statments
//     - `--binlog-row-image=FULL`: before and after image of row changes
//     - `--binlog-row-metadata=FULL`: extra optional meta for tables such as signedness for numeric columns/column names ...
//
// ref:
//   - https://mysqlhighavailability.com/more-metadata-is-written-into-binary-log/
//   - https://mysqlhighavailability.com/taking-advantage-of-new-transaction-length-metadata/
//   - https://github.com/siddontang/go-mysql/pull/468
//
// Compatiable between fulldump/incrdump (see tst):
//   - DECIMAL/NUMERIC fields are returned as string, but may have different trailing zeros.
//   - BINARY fields are returned as string, but may have different trailing '\x00'.
//   - JSON fields are returned as string, but elements inside may have different position.
//   - BIT fields seems have some bugs in incrdump.
//   - TIME with fraction seems have some bugs in incrdump too.
package mycanal
