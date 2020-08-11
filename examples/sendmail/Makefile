SHELL:=/bin/bash

gen:
	protoc --go_out=paths=source_relative:. --nproto_out=paths=source_relative:. *.proto

clean:
	rm -f *.pb.go *.nproto.go
