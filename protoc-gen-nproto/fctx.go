package main

import (
	"fmt"

	pgs "github.com/lyft/protoc-gen-star"
)

type FileContext struct {
	mod      *NProtoModule
	curPath  pgs.FilePath              // Import path for current file.
	path2pkg map[pgs.FilePath]pgs.Name // Import path -> package name (alias)
	pkg2path map[pgs.Name]pgs.FilePath // Package name (alias) -> import path
}

func NewFileContext(mod *NProtoModule, curFile pgs.File) *FileContext {
	return &FileContext{
		mod:      mod,
		curPath:  mod.ctx.ImportPath(curFile),
		path2pkg: make(map[pgs.FilePath]pgs.Name),
		pkg2path: make(map[pgs.Name]pgs.FilePath),
	}
}

func (fctx *FileContext) Path2Pkg() map[pgs.FilePath]pgs.Name {
	return fctx.path2pkg
}

func (fctx *FileContext) Pkg2Path() map[pgs.Name]pgs.FilePath {
	return fctx.pkg2path
}

func (fctx *FileContext) Import(pkg, path string) string {
	fctx.ImportPkg(pgs.Name(pkg), pgs.FilePath(path))
	return ""
}

func (fctx *FileContext) Imported(path pgs.FilePath) bool {
	// Always returns true for current package.
	if path == fctx.curPath {
		return true
	}
	_, imported := fctx.path2pkg[path]
	return imported
}

func (fctx *FileContext) ImportPkg(pkg pgs.Name, path pgs.FilePath) {
	// Already imported.
	if fctx.Imported(path) {
		return
	}

	// Check pkg duplication. Adds a number suffix if duplicated.
	if _, dup := fctx.pkg2path[pkg]; dup {
		suffix := 2
		pkgAlias := pkg
		for {
			pkgAlias = pgs.Name(fmt.Sprintf("%s%d", pkg, suffix))
			if _, dup := fctx.pkg2path[pkgAlias]; !dup {
				break
			}
			suffix += 1
		}
		pkg = pkgAlias
	}

	// Add to two maps.
	fctx.path2pkg[path] = pkg
	fctx.pkg2path[pkg] = path
}

func (fctx *FileContext) PackagePrefixed(entity pgs.Entity) string {
	// If the entity's package is the same as current file.
	// No prefix is needed.
	ctx := fctx.mod.ctx
	path := ctx.ImportPath(entity)
	name := ctx.Name(entity)
	if path == fctx.curPath {
		return string(name)
	}

	// Import it if not imported.
	if !fctx.Imported(path) {
		fctx.ImportPkg(ctx.PackageName(entity), path)
	}

	// Returns the prefixed name.
	return fmt.Sprintf("%s.%s", fctx.path2pkg[path], name)
}
