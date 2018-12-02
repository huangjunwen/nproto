all:
	protoc --go_out=paths=source_relative:. --nproto_out=paths=source_relative:. *.proto && gofmt -w *.go

clean:
	rm *.go
