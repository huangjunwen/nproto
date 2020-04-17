# NProto

[![GoDoc](https://godoc.org/github.com/huangjunwen/nproto?status.svg)](http://godoc.org/github.com/huangjunwen/nproto)
[![Go Report](https://goreportcard.com/badge/github.com/huangjunwen/nproto)](https://goreportcard.com/report/github.com/huangjunwen/nproto)
[![Build Status](https://travis-ci.org/huangjunwen/nproto.svg?branch=master)](https://travis-ci.org/huangjunwen/nproto) 
[![codecov](https://codecov.io/gh/huangjunwen/nproto/branch/master/graph/badge.svg)](https://codecov.io/gh/huangjunwen/nproto)

Some easy to use rpc/msg components.

## Components

- [x] RPC server/client using [nats](https://github.com/nats-io/nats-server) as transport with json/protobuf encoding: [natsrpc](https://godoc.org/github.com/huangjunwen/nproto/nproto/natsrpc)
- [x] Auto reconnection/resubscription client for [nats-streaming](https://github.com/nats-io/nats-streaming-server): [stanmsg](https://godoc.org/github.com/huangjunwen/nproto/nproto/stanmsg)
- [x] Pipeline msgs from RDBMS to downstream publisher (*deprecating*): [dbpipe](https://godoc.org/github.com/huangjunwen/nproto/nproto/dbpipe)
- [x] Pipeline msgs from MySQL8 binlog to downstream publisher: [binlogmsg](https://godoc.org/github.com/huangjunwen/nproto/nproto/binlogmsg)
- [x] Opentracing support: [tracing](https://godoc.org/github.com/huangjunwen/nproto/nproto/tracing)
- [x] Structure (json) logging support using [zerolog](https://github.com/rs/zerolog): [zlog](https://godoc.org/github.com/huangjunwen/nproto/nproto/zlog)
- [x] Protoc plugin to generate stub code for above components: protoc-gen-nproto

## Helper packages that can be used standalone
- [x] Task runner to contorl resource usage: [taskrunner](https://godoc.org/github.com/huangjunwen/nproto/helpers/taskrunner)
- [x] CDC (Change Data Capture) for MySQL8+: [mycanal](https://godoc.org/github.com/huangjunwen/nproto/helpers/mycanal)

## Install

You need to install [protobuf's compiler](https://github.com/protocolbuffers/protobuf/releases) and [protoc-gen-go](https://github.com/golang/protobuf) first.

To install libraries, run: `$ go get -u github.com/huangjunwen/nproto/nproto/...`

To install the protoc plugin, run: `$ go get -u github.com/huangjunwen/nproto/protoc-gen-nproto`

## Usage

1. As always you needs to write a proto file, for example:

```protobuf
syntax = "proto3";
package huangjunwen.nproto.tests.mathapi;

option go_package = "github.com/huangjunwen/nproto/tests/math/api;mathapi";

message SumRequest {
  repeated double args = 1;
}

message SumReply {
  double sum = 1;
}

// Math is a service providing some math functions.
// @@nprpc@@
service Math {
  // Sum returns the sum of a list of arguments.
  rpc Sum(SumRequest) returns (SumReply);
}
```

2. If you want `protoc-gen-nproto` to generate stub code for your service to use with rpc components, add `@@nprpc@@` at any position in the leading comment of the service.

3. Likewise, if you want `protoc-gen-nproto` to generate stub code for your message to use with msg components, add `@@npmsg@@` at any position in the leading comment of the message.

4. Run `protoc-gen-nproto`, for example:

```bash
$ protoc --go_out=paths=source_relative:. --nproto_out=paths=source_relative:. *.proto
```

5. Implement your service/message handler, then glue them with the stub code generated, see [this simple exmaple](https://github.com/huangjunwen/nproto/tree/master/tests/math) for detail.
