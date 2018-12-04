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

const fakePkgClause = "package xxxxxxxx\n\n"

func (p GoFmt) Process(in []byte) ([]byte, error) {
	// Contains 'package xxx' or not?
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", in, parser.PackageClauseOnly)
	hasPkgClause := err == nil

	// If has package clause.
	if hasPkgClause {
		return format.Source(in)
	}

	// Otherwise add a fake package clause.
	input := []byte(fakePkgClause)
	input = append(input, in...)

	// Format.
	output, err := format.Source(input)
	if err != nil {
		return nil, err
	}

	// Trim the fake package clause.
	output = output[len(fakePkgClause):]

	// Add one more new line.
	output = append(output, '\n')
	return output, nil
}
