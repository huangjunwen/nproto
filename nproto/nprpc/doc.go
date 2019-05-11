// Package nprpc contains NatsRPCServer/NatsRPCClient which implement nproto.RPCServer and nproto.RPCClient.
//
// Servers/clients with same subject prefix within a same nats cluster form a RPC namespace.
//
// Metadata (nproto.MD) attached in client (outgoing) side will be passed unmodified to server (incoming) side.
package nprpc
