package main

import (
	"text/template"

	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
)

type NProtoModule struct {
	pgs.ModuleBase
	ctx         pgsgo.Context
	funcs       map[string]interface{}
	skeletonTpl *template.Template
	nprpcTpl    *template.Template
}

func (m *NProtoModule) InitContext(c pgs.BuildContext) {
	m.ModuleBase.InitContext(c)
	m.ctx = pgsgo.InitContext(c.Parameters())
	m.funcs = map[string]interface{}{
		"PackageName": m.ctx.PackageName,
		"Name": func(node pgs.Node) pgs.Name {
			switch n := node.(type) {
			case pgs.Service:
				// NOTE: We don't need the "Server" suffix.
				return pgsgo.PGGUpperCamelCase(n.Name())
			default:
				return m.ctx.Name(node)
			}
		},
		"ImportPath": m.ctx.ImportPath,
	}
	m.skeletonTpl = m.newTpl("skeleton", skeletonTplText)
	m.nprpcTpl = m.newTpl("nprpc", nprpcTplText)
}

func (m *NProtoModule) newTpl(name, text string) *template.Template {
	return template.Must(template.New(name).Funcs(m.funcs).Parse(text))
}

func (m *NProtoModule) Name() string {
	return "nproto"
}

func (m *NProtoModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	// For each target file.
	for _, target := range targets {
		targetName := m.ctx.OutputPath(target).SetExt(".nproto.go").String()
		m.AddGeneratorTemplateFile(targetName, m.skeletonTpl, map[string]interface{}{
			"File": target,
		})

		for _, service := range target.Services() {
			m.AddGeneratorTemplateInjection(targetName, "nprpc", m.nprpcTpl, map[string]interface{}{
				"Service": service,
			})
		}
	}
	return m.Artifacts()
}

const (
	skeletonTplText = `
package {{ PackageName .File }}

import (
// @@protoc_insertion_point(imports)
)

// @@protoc_insertion_point(nprpc)

// @@protoc_insertion_point(npmsg)
`
	nprpcTplText = `
/*{{ .Service.SourceCodeInfo.LeadingComments }}*/
type {{ Name .Service }} interface {
	// TODO
}

`
)
