# NProto

nats (nats.io) + protobuf

## Features

- RPC over nats
- Durable message over nats-streaming

## Install

Depends on protobuf and [protoc-gen-gotemplate](https://github.com/moul/protoc-gen-gotemplate)

- Install protobuf: https://github.com/protocolbuffers/protobuf/releases
- Install protoc-gen-go: `go get -u github.com/golang/protobuf/protoc-gen-go`
- Install protoc-gen-gotemplate:
  ```
  # Use my fork (with some modification, still not merged)

  $ mkdir -p $GOPATH/src/github.com/moul/protoc-gen-gotemplate
  $ cd $GOPATH/src/github.com/moul/protoc-gen-gotemplate
  $ git clone https://github.com/huangjunwen/protoc-gen-gotemplate.git .
  $ go get github.com/moul/protoc-gen-gotemplate
  ```

## Usage

```bash
$ $GOPATH/src/github.com/huangjunwen/nproto/nproto.sh <proto files directory>

```
