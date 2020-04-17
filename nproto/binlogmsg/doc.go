// Package binlogmsg contains BinlogMsgPipe which is used as a publisher pipeline from MySQL-8 to downstream publisher.
//
// Typical workflow:
//   - `CreateMsgTable` creates a msg table to store msgs.
//   - `NewBinlogMsgPipe` creates a pipe to connect MySQL binlog and downstream publisher.
//   - `NewBinlogMsgPublisher` then publish messages.
package binlogmsg
