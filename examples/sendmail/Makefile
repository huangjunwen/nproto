SHELL:=/bin/bash

all:
	cd server && make && cd ../client && make

gen:
	protoc --go_out=paths=source_relative:. --nproto2_out=paths=source_relative:. *.proto

clean:
	rm -f *.pb.go *.nproto.go
