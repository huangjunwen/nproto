// Package mycanal is a CDC (Change Data Capture) library for MySQL8+
//
// It provides helper functions for full dump (package fulldump) and incremental change capture (package incrdump).
// Prerequisites:
//   - MySQL-8.0.2 and above
//   - GTID mode enabled:
//     - `--gtid-mode=ON`
//     - `--enforce-gtid-consistency=ON`
//   - binlog enabled with the following:
//     - `--binlog-format=ROW`: binlog output row changes instead of statments
//     - `--binlog-row-image=FULL`: before and after image of row changes
//     - `--binlog-row-metadata=FULL`: extra optional meta for tables such as signedness for numeric columns/column names ...
//
// ref:
//   - https://mysqlhighavailability.com/more-metadata-is-written-into-binary-log/
//   - https://mysqlhighavailability.com/taking-advantage-of-new-transaction-length-metadata/
//   - https://github.com/siddontang/go-mysql/pull/468
package mycanal
