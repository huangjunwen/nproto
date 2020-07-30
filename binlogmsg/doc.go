// Package binlogmsg contains publisher implemenation to 'publish' (store) messages to MySQL8
// tables then flush to downstream publisher using binlog notification.
//
// Since messages are stored in normal MySQL tables, all ACID properties are applied to them.
package binlogmsg
