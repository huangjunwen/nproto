package main

import (
	"fmt"

	pgs "github.com/lyft/protoc-gen-star"
)

// FileContext contains context information of a file during code generation.
type FileContext struct {
	mod      *NProtoModule
	curPath  pgs.FilePath              // Import path for current file.
	path2pkg map[pgs.FilePath]pgs.Name // Import path -> package name (alias)
	pkg2path map[pgs.Name]pgs.FilePath // Package name (alias) -> import path
}

// NewFileContext creates a new FileContext.
func NewFileContext(mod *NProtoModule, curFile pgs.File) *FileContext {
	return &FileContext{
		mod:      mod,
		curPath:  mod.ctx.ImportPath(curFile),
		path2pkg: make(map[pgs.FilePath]pgs.Name),
		pkg2path: make(map[pgs.Name]pgs.FilePath),
	}
}

// Path2Pkg returns the mapping: import path -> package name (alias).
func (fctx *FileContext) Path2Pkg() map[pgs.FilePath]pgs.Name {
	return fctx.path2pkg
}

// Pkg2Path returns the mapping: package name (alias) -> import path.
func (fctx *FileContext) Pkg2Path() map[pgs.Name]pgs.FilePath {
	return fctx.pkg2path
}

// Import is used in template to declare an 'import'.
func (fctx *FileContext) Import(pkg, path string) string {
	fctx.ImportPkg(pgs.Name(pkg), pgs.FilePath(path))
	return ""
}

// Imported returns true if a package has been imported. The current package is always imported.
func (fctx *FileContext) Imported(path pgs.FilePath) bool {
	// Always returns true for current package.
	if path == fctx.curPath {
		return true
	}
	_, imported := fctx.path2pkg[path]
	return imported
}

// ImportPkg is used to import a package into the FileContext.
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

// PackagePrefixed is used in template to prefix an entity with its package name, e.g. `timestamp.Timestamp`.
// It will also import the package into FileContext.
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
