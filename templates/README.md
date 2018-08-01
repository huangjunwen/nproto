# templates

Use [protoc-gen-gotemplate](https://github.com/moul/protoc-gen-gotemplate) to generate stub code.

Usage:

- Each `.proto` file should contain at most one service.
- `.proto` files should follow style guide: https://developers.google.com/protocol-buffers/docs/style
- Set `single-package-mode` to true.

Example:

```bash
$ protoc --go_out=. --gotemplate_out=single-package-mode=true,template_dir=$GOPATH/src/github.com/huangjunwen/nproto/templates:. *.proto && gofmt -w *.go

```
