package main

import (
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	pgs "github.com/lyft/protoc-gen-star"
)

type GoFmt struct{}

func (p GoFmt) Match(a pgs.Artifact) bool {
	var n string

	switch a := a.(type) {
	case pgs.GeneratorFile:
		n = a.Name
	case pgs.GeneratorTemplateFile:
		n = a.Name
	case pgs.CustomFile:
		n = a.Name
	case pgs.CustomTemplateFile:
		n = a.Name
	case pgs.GeneratorInjection:
		n = a.FileName
	case pgs.GeneratorTemplateInjection:
		n = a.FileName
	default:
		return false
	}

	return strings.HasSuffix(n, ".go")
}

const fakePkgClause = "package xxxxxxxx\n"

func (p GoFmt) Process(in []byte) ([]byte, error) {
	// Contains 'package xxx' or not?
	input := in
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", in, parser.PackageClauseOnly)
	hasPkgClause := err == nil

	// Add a fake package clause if missing.
	if !hasPkgClause {
		input = make([]byte, 0, len(fakePkgClause)+len(in))
		input = append(input, fakePkgClause...)
		input = append(input, in...)
	}

	// Format.
	output, err := format.Source(input)
	if err != nil {
		return nil, err
	}

	// Trim the fake package clause if any.
	if !hasPkgClause {
		return output[len(fakePkgClause):], nil
	}
	return output, nil
}
