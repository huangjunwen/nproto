// Package dbpipe contains DBMsgPublisherPipe which is used as a publisher pipeline from RDBMS to downstream publisher (deprecating).
//
// Typical workflow (ignore error handling):
//
//   // Creates DBMsgPublisherPipe.
//   // `downstream` specifies the sink of pipeline.
//   // `dialect` specifies the dialect of RDBMS.
//   // `db`/`table` specify which table to store message data.
//   pipe, _ := dbpipe.NewDBMsgPublisherPipe(downstream, dialect, db, table)
//
//   // Run transaction.
//   sqlh.WithTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
//       // Create a new publisher inside transaction.
//       publisher := pipe.NewMsgPublisherWithTx(ctx, tx)
//
//       // ... Some db operations ....
//
//       publisher.Publish(ctx, "subject", "data")
//
//       // ... Some more db operations ...
//
//       return nil
//   })
//
package dbpipe
