// Package durconn contains DurConn which implements nproto.MsgAsyncPublisher and nproto.MsgSubscriber.
//
// DurConn is a "durable" connection to nats streaming server which handles reconnecting and resubscription automatically.
//
// DurConn(s) with same subject prefix within a same nats streaming cluster form a message delivery namespace.
//
// Metadata (nproto.MD) attached in publisher (outgoing) side will be passed unmodified to subscriber (incoming) side.
//
// To graceful shutdown DurConn, one should first wait task runner to finish all ongoing handlers then close the DurConn.
// Since nats-streaming-server will redeliver messages that are not handled after task runner finished.
package durconn
