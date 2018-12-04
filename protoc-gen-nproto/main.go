package main

import (
	pgs "github.com/lyft/protoc-gen-star"
)

func main() {
	pgs.Init().RegisterModule(
		&NProtoModule{},
	).RegisterPostProcessor(
		GoFmt{},
	).Render()
}
