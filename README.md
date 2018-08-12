# nproto
nats + protobuf tools

## Install

Depends on protobuf and [protoc-gen-gotemplate](https://github.com/moul/protoc-gen-gotemplate)

- Install protobuf: `go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`
- Install protoc-gen-gotemplate:
  ```
  # Use my fork (with some modification, still not merged)

  $ mkdir -p $GOPATH/src/github.com/moul/protoc-gen-gotemplate
  $ cd $GOPATH/src/github.com/moul/protoc-gen-gotemplate
  $ git clone https://github.com/huangjunwen/protoc-gen-gotemplate.git .
  $ go get github.com/moul/protoc-gen-gotemplate
  ```
