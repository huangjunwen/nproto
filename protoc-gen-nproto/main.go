package main

import (
	pgs "github.com/lyft/protoc-gen-star"
)

func main() {
	pgs.Init().RegisterModule(
		&nprotoModule{},
	).RegisterPostProcessor(
		goFmt{},
	).Render()
}
