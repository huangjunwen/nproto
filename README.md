# NProto

Some easy to use rpc/msg components.

This is the second major version (v2). Compare to v1, v2 has re-designed high level interfaces:
Though it's still mainly focus on [nats](https://nats.io) + [protobuf](https://developers.google.com/protocol-buffers),
but it's not force to. It's totally ok for implementations to use other encoding schemas (e.g. `json`, `msgpack` ...) 
or use other transports (e.g. `http`).
