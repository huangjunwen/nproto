# NProto

Easy to use communication (e.g. rpc/msg/...) components.

This is the second major version (v2). Compare to v1, v2 has re-designed high level interfaces:
Though it's still mainly focus on [**n**ats](https://nats.io) + [**proto**buf](https://developers.google.com/protocol-buffers),
but it's not force to. It's totally ok for implementations to use other encoding schemas (e.g. `json`, `msgpack` ...) 
or use other transports (e.g. `http`).

## Packages
- [x] [rpc](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/rpc?tab=doc): High level types/interfaces for rpc implementations.
- [x] [msg](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/msg?tab=doc): High level types/interfaces for msg implementations.
- [x] [md](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/md?tab=doc): Meta data types.
- [x] [enc](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/enc?tab=doc): Data encoding/decoding type.
- [x] [natsrpc](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/natsrpc?tab=doc): Rpc implementation using nats as transport.
- [x] [stanmsg](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/stanmsg?tab=doc): Auto reconnection/resubscription client for nats-streaming and msg implementation.
- [x] [binlogmsg](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/binlogmsg?tab=doc): 'Publish' (store) messages to MySQL8 tables then flush to downstream publisher using binlog notification.
- [x] [tracing](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/tracing?tab=doc): Opentracing middlewares.
- [x] [protoc-gen-nproto](https://pkg.go.dev/github.com/huangjunwen/nproto/v2/protoc-gen-nproto?tab=doc): Stub code generator for protobuf.
