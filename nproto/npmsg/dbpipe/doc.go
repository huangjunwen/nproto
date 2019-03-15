// Package dbpipe contains DBMsgPublisherPipe which is used as a publisher pipeline from RDBMS to downstream publisher.
//
// Typical workflow (ignore error handling):
//
//    // Creates DBMsgPublisherPipe.
//    // `downstream` specifies the sink of pipeline.
//    // `dialect` specifies the dialect of RDBMS.
//    // `db`/`table` specify which table to store message data.
//    pipe, _ := dbpipe.NewDBMsgPublisherPipe(downstream, dialect, db, table)
//
//    // Starts a transaction.
//    tx, _ := db.BeginTx()
//    defer tx.Rollback()
//
//    // Creates a MsgPublisher for this transaction.
//    publisher := pipe.NewMsgPublisher(tx)
//
//    // ... Some db operations ...
//
//    // Publishes message. The message (as well as outgoing meta data attached to `ctx`)
//    // will be saved to database. Note that the message can be rollbacked.
//    publisher.Publish(ctx, "subject", "data")
//
//    // ... Some more db operations ...
//
//    if everythingIsOK {
//       tx.Commit()
//       // If the transaction has been committed successfully. Flush messages
//       // to downstream. There is also a background redelivery loop in DBMsgPublisherPipe
//       // to flush periodically to ensure committed messages publishing to downstream at
//       // least once.
//       publisher.Flush(ctx)
//    }
package dbpipe
