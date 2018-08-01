# templates

Use [protoc-gen-gotemplate](https://github.com/moul/protoc-gen-gotemplate) to generate stub code, example:

```bash
$ protoc --go_out=. --gotemplate_out=single-package-mode=true,template_dir=$GOPATH/src/github.com/huangjunwen/nproto/templates:. *.proto && gofmt -w *.go

```

**NOTE: proto files should follow the style guide: https://developers.google.com/protocol-buffers/docs/style**
