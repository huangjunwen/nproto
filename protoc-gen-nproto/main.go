package main

import (
	"fmt"
	"os"
	"runtime/debug"

	pgs "github.com/lyft/protoc-gen-star"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("protoc-gen-nproto version=v? sum=?")
		} else {
			fmt.Printf("protoc-gen-nproto version=%s sum=%s\n", info.Main.Version, info.Main.Sum)
		}
		return
	}
	pgs.Init().RegisterModule(
		&nprotoModule{},
	).RegisterPostProcessor(
		goFmt{},
	).Render()
}
