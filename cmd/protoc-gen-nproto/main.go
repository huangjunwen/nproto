package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"google.golang.org/protobuf/compiler/protogen"
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

	protogen.Options{}.Run(func(p *protogen.Plugin) error {
		for _, f := range p.Files {
			if f.Generate {
				genFile(p, f)
			}
		}
		return nil
	})
}
