# NProto

[![GoDoc](https://godoc.org/github.com/huangjunwen/nproto?status.svg)](http://godoc.org/github.com/huangjunwen/nproto)
[![Go Report](https://goreportcard.com/badge/github.com/huangjunwen/nproto)](https://goreportcard.com/report/github.com/huangjunwen/nproto)
[![Build Status](https://travis-ci.org/huangjunwen/nproto.svg?branch=master)](https://travis-ci.org/huangjunwen/nproto) 
[![codecov](https://codecov.io/gh/huangjunwen/nproto/branch/master/graph/badge.svg)](https://codecov.io/gh/huangjunwen/nproto)

Some easy to use components using [NATS](https://nats.io) ([gnatsd](https://github.com/nats-io/gnatsd)/[nats-streaming-server](https://github.com/nats-io/nats-streaming-server)) with [protocol-buffer](https://developers.google.com/protocol-buffers/).

## Why

> NATS Server is a simple, high performance open source messaging system ....

Simple is very important to me. With a single deployment, one gets a high performance message broker with rpc/service discovery/load balancing/at most(or at least) once message delivery/... all at once.

## Components

- [x] RPC server/client using nats as transport with json/protobuf encoding: [nprpc](https://godoc.org/github.com/huangjunwen/nproto/nproto/nprpc)
- [x] Auto reconnection/resubscription client for nats-streaming: [durconn](https://godoc.org/github.com/huangjunwen/nproto/nproto/npmsg/durconn)
- [x] Pipeline msgs from RDBMS to downstream publisher: [dbpipe](https://godoc.org/github.com/huangjunwen/nproto/nproto/npmsg/dbpipe)
- [x] Protoc plugin to generate stub code for above components: protoc-gen-nproto
- [x] Task runner to contorl resource usage: [taskrunner](https://godoc.org/github.com/huangjunwen/nproto/nproto/taskrunner)
- [x] Opentracing support: [tracing](https://godoc.org/github.com/huangjunwen/nproto/nproto/tracing)

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

2. If you want `protoc-gen-nproto` to generate stub code for your service to use with nprpc, add `@@nprpc@@` at any position in the leading comment of the service.

3. Likewise, if you want `protoc-gen-nproto` to generate stub code for your message to use with npmsg, add `@@npmsg@@` at any position in the leading comment of the message.

4. Run `protoc-gen-nproto`, for example:

```bash
$ protoc --go_out=paths=source_relative:. --nproto_out=paths=source_relative:. *.proto
```

5. Implement your service/message handler, then glue them with the stub code generated, see [this simple exmaple](https://github.com/huangjunwen/nproto/tree/master/tests/math) for detail.
