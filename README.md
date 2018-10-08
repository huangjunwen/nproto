# NProto

nats (nats.io) + protobuf

## Features

- RPC over nats
- Durable message over nats-streaming

## Install

Depends on protobuf and [protoc-gen-gotemplate](https://github.com/moul/protoc-gen-gotemplate)

- Install protobuf: https://github.com/protocolbuffers/protobuf/releases
- Install protoc-gen-go: `go get -u github.com/golang/protobuf/protoc-gen-go`
- Install my fork (with some modification but still not merged) of protoc-gen-gotemplate: `go get -u github.com/huangjunwen/protoc-gen-gotemplate`

## Usage

```bash
$ $GOPATH/src/github.com/huangjunwen/nproto/nproto.sh <proto files directory>

```
