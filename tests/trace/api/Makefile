all:
	protoc --go_out=paths=source_relative:. --nproto_out=paths=source_relative:. *.proto 

clean:
	rm math.pb.go math.nproto.go
