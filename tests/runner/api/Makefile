all:
	protoc --go_out=paths=source_relative:. --nproto_out=paths=source_relative:. *.proto 

clean:
	rm *.pb.go *.nproto.go
