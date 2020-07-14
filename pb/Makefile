PB_DIR=$(realpath $(dir $(realpath $(lastword $(MAKEFILE_LIST)))))

gen:
	find . -mindepth 1 -type d -print | xargs -I {} bash -c \
		'cd {} && protoc -I . -I ${PB_DIR} --go_out=paths=source_relative:. *.proto'

clean:
	find . -mindepth 1 -type d -print | xargs -I {} bash -c \
		'cd {} && rm *.pb.go'
