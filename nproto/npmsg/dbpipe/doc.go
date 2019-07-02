// Package dbpipe contains DBMsgPublisherPipe which is used as a publisher pipeline from RDBMS to downstream publisher.
//
// Typical workflow (ignore error handling):
//
//   // Creates DBMsgPublisherPipe.
//   // `downstream` specifies the sink of pipeline.
//   // `dialect` specifies the dialect of RDBMS.
//   // `db`/`table` specify which table to store message data.
//   pipe, _ := dbpipe.NewDBMsgPublisherPipe(downstream, dialect, db, table)
//
//   // Creates a TxPlugin.
//   txp := pipe.TxPlugin()
//
//   // Run transaction with txp.
//   sqlh.WithTx(db, func(tx *sql.Tx) error {
//       // Get the publisher inside transaction.
//       publisher := txp.Publisher()
//
//       // ... Some db operations ....
//
//       publisher.Publish(ctx, "subject", "data")
//
//       // ... Some more db operations ...
//
//       return nil
//   }, txp)
//
package dbpipe
