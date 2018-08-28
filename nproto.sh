#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage:"
  echo "  nproto.sh <proto-directory>"
  exit 1
fi

if [ ! -d "$1" ]; then
  echo "\"$1\" is not a directory"
  exit 1
fi

protoc --go_out=$1 --gotemplate_out=all=true,single-package-mode=true,template_dir=$GOPATH/src/github.com/huangjunwen/nproto/templates:. $1/*.proto && gofmt -w $1/*.go
