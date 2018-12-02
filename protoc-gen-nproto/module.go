package main

import (
	"log"
	"strings"
	"text/template"

	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
)

type NProtoModule struct {
	pgs.ModuleBase
	ctx         pgsgo.Context
	skeletonTpl *template.Template
	importsTpl  *template.Template
	nprpcTpl    *template.Template
}

func (m *NProtoModule) InitContext(c pgs.BuildContext) {
	m.ModuleBase.InitContext(c)
	m.ctx = pgsgo.InitContext(c.Parameters())

	funcs := map[string]interface{}{
		"Name": func(node pgs.Node) pgs.Name {
			switch n := node.(type) {
			case pgs.Service:
				// NOTE: We don't need the "Server" suffix.
				return pgsgo.PGGUpperCamelCase(n.Name())
			default:
				return m.ctx.Name(node)
			}
		},
		"PackageName": m.ctx.PackageName,
		"ImportPath":  m.ctx.ImportPath,
		"LeadingComments": func(entity pgs.Entity) string {
			comments := strings.TrimSpace(entity.SourceCodeInfo().LeadingComments())
			if comments == "" {
				return ""
			}
			lines := strings.Split(comments, "\n")
			for i, line := range lines {
				lines[i] = "//" + line
			}
			return strings.Join(lines, "\n")
		},
	}
	newTpl := func(name, text string) *template.Template {
		return template.Must(template.New(name).Funcs(funcs).Parse(text))
	}
	m.skeletonTpl = newTpl("skeleton", skeletonTplText)
	m.importsTpl = newTpl("imports", importsTplText)
	m.nprpcTpl = newTpl("nprpc", nprpcTplText)
}

func (m *NProtoModule) Name() string {
	return "nproto"
}

func (m *NProtoModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	// For each target file.
	for _, target := range targets {
		// Replace ".pb.go" to "nproto.go"
		targetName := strings.TrimSuffix(string(m.ctx.OutputPath(target)), ".pb.go") + ".nproto.go"

		// Create file context for target.
		fctx := NewFileContext(m, target)

		// Create target file with skeleton.
		m.AddGeneratorTemplateFile(targetName, m.skeletonTpl, map[string]interface{}{
			"File":        target,
			"FileContext": fctx,
		})

		// Insert nprpc stubs.
		for _, service := range target.Services() {
			for _, method := range service.Methods() {
				if method.ClientStreaming() || method.ServerStreaming() {
					log.Fatal("nproto.nprpc does not support streaming input/output")
				}
			}
			m.AddGeneratorTemplateInjection(targetName, "nprpc", m.nprpcTpl, map[string]interface{}{
				"Service":     service,
				"FileContext": fctx,
			})
		}

		// TODO

		// Insert imports.
		m.AddGeneratorTemplateInjection(targetName, "imports", m.importsTpl, map[string]interface{}{
			"FileContext": fctx,
		})
	}
	return m.Artifacts()
}

const (
	skeletonTplText = `
package {{ PackageName .File }} // import {{ printf "%+q" (ImportPath .File) }}

// @@protoc_insertion_point(imports)

// @@protoc_insertion_point(nprpc)

// @@protoc_insertion_point(npmsg)
`

	importsTplText = `
{{- $fctx := .FileContext -}}
import (
	{{- $fctx.Import "context" "context" -}}
	{{- $fctx.Import "proto" "github.com/golang/protobuf/proto" -}}
	{{- $fctx.Import "nproto" "github.com/huangjunwen/nproto/nproto" -}}
	{{ range $pkg, $path := .FileContext.Pkg2Path -}}
		{{ $pkg }} {{ printf "%+q" $path }}
	{{ end }}
)

var (
	_ = context.Background
	_ = proto.Int
	_ = nproto.NewRPCCtx
)

`
	nprpcTplText = `
{{- $svc := .Service -}}
{{- $fctx := .FileContext -}}

{{ LeadingComments $svc }}
type {{ Name $svc }} interface {
	{{ range $svc.Methods }}
		{{ LeadingComments . }}
		{{ Name . }}(ctx context.Context, input *{{ $fctx.PackagePrefixed .Input }}) (output *{{ $fctx.PackagePrefixed .Output }}, err error)
	{{ end }}
}

// Serve{{ Name $svc }} use rpc server to serve {{ Name $svc }}.
func Serve{{ Name $svc }}(server nproto.RPCServer, svcName string, svc {{ Name $svc }}) error {
	return server.RegistSvc(svcName, map[*nproto.RPCMethod]nproto.RPCHandler{
		{{ range $svc.Methods -}}
			method{{ Name $svc }}__{{ Name . }}: func(ctx context.Context, input proto.Message) (proto.Message, error) {
				return svc.{{ Name . }}(ctx, input.(*{{ $fctx.PackagePrefixed .Input }}))
			},
		{{ end -}}
	})
}

// Invoke{{ Name $svc }} use rpc client to invoke {{ Name $svc }} service.
func Invoke{{ Name $svc }}(client nproto.RPCClient, svcName string) {{ Name $svc }} {
	return &client{{ Name $svc }}{
		{{ range $svc.Methods -}}
			handler{{ Name . }}: client.MakeHandler(svcName, method{{ Name $svc }}__{{ Name . }}),
		{{ end }}
	}
}

{{ range $svc.Methods -}}
var method{{ Name $svc }}__{{ Name . }} = &nproto.RPCMethod{
	Name: "{{ Name . }}",
	NewInput: func() proto.Message { return &{{ $fctx.PackagePrefixed .Input }}{} },
	NewOutput: func() proto.Message { return &{{ $fctx.PackagePrefixed .Output }}{} },
}
{{ end }}

type client{{ Name $svc }} struct {
	{{ range $svc.Methods -}}
		handler{{ Name . }} nproto.RPCHandler
	{{ end -}}
}

{{ range $svc.Methods -}}
	// {{ Name . }} implements {{ Name $svc }} interface.
	func (svc *client{{ Name $svc }}) {{ Name . }}(ctx context.Context, input *{{ $fctx.PackagePrefixed .Input }}) (*{{ $fctx.PackagePrefixed .Output }}, error) {
		output, err := svc.handler{{ Name . }}(ctx, input)
		if err != nil {
			return nil, err
		}
		return output.(*{{ $fctx.PackagePrefixed .Output }}), nil
	}
{{ end }}

`
)
