package main

import (
	"log"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
)

const (
	genRPCStubLabel = "@@nprpc@@"
	genMsgStubLabel = "@@npmsg@@"
)

func genFile(p *protogen.Plugin, f *protogen.File) {
	g := p.NewGeneratedFile(f.GeneratedFilenamePrefix+".nproto.go", f.GoImportPath)

	g.P("package ", f.GoPackageName)
	g.P()

	for _, service := range f.Services {
		if strings.Contains(string(service.Comments.Leading), genRPCStubLabel) {
			genRPCStub(g, f, service)
		}
	}

	for _, message := range f.Messages {
		if strings.Contains(string(message.Comments.Leading), genMsgStubLabel) {
			genMsgStub(g, f, message)
		}
	}

}

func genRPCStub(g *protogen.GeneratedFile, f *protogen.File, service *protogen.Service) {

	funcs := map[string]interface{}{
		"ident":        g.QualifiedGoIdent,
		"contextIdent": contextIdent,
		"nprpcIdent":   nprpcIdent,
	}

	rpcStubTpl, err := template.New("rpc").Funcs(funcs).Parse(`
// --- RPC stub for {{ .GoName }} ---

{{ .Comments.Leading.String -}}
type {{ .GoName }} interface {
	{{- range .Methods }}
		{{ .Comments.Leading.String -}}
		{{ .GoName }}({{ ident (contextIdent "Context")}}, *{{ ident .Input.GoIdent }}) (*{{ ident .Output.GoIdent }}, error)
	{{ end }}
}

// {{ .GoName }}Spec contains method specs of service {{ .GoName }}.
type {{ .GoName }}Spec struct {
	{{- range .Methods }}
		spec{{ .GoName }} {{ ident (nprpcIdent "RPCSpec") }}
	{{- end }}
}

type client{{ .GoName }} struct {
	{{- range .Methods }}
		handler{{ .GoName }} {{ ident (nprpcIdent "RPCHandler") }}
	{{- end }}
}

// New{{ .GoName }}Spec creates a new {{ .GoName }}Spec.
func New{{ .GoName }}Spec(svcName string) *{{ .GoName }}Spec {
	return &{{ .GoName }}Spec{
		{{- range .Methods }}
			spec{{ .GoName }}: {{ ident (nprpcIdent "MustRPCSpec") }}(
				svcName,
				"{{ .GoName }}",
				func() interface{} { return new({{ ident .Input.GoIdent }}) },
				func() interface{} { return new({{ ident .Output.GoIdent }}) },
			),
		{{- end}}
	}
}

// SpecMap returns a mapping from method name to method spec.
func (spec *{{ .GoName }}Spec) SpecMap() map[string]{{ ident (nprpcIdent "RPCSpec") }} {
	return map[string]{{ ident (nprpcIdent "RPCSpec") }}{
		{{- range .Methods}}
			"{{ .GoName }}": spec.spec{{ .GoName }},
		{{- end}}
	}
}

// Serve{{ .GoName }} serves {{ .GoName }} service using a RPCServer.
func Serve{{ .GoName }}(server {{ ident (nprpcIdent "RPCServer") }}, svcSpec *{{ .GoName }}Spec, svc {{ .GoName }}) (err error) {
	{{- range .Methods }}
		if err = server.RegistHandler(
			svcSpec.spec{{ .GoName }}, 
			func(ctx {{ ident (contextIdent "Context") }}, input interface{}) (interface{}, error) {
				return svc.{{ .GoName }}(ctx, input.(*{{ ident .Input.GoIdent }}))
			},
		); err != nil {
			return err
		}
		defer func() {
			if err != nil {
				// Deregist if error.
				server.RegistHandler(svcSpec.spec{{ .GoName }}, nil)
			}
		}()
	{{ end }}
	return nil
}

// Invoke{{ .GoName }} invokes {{ .GoName }} service using a RPCClient.
func Invoke{{ .GoName }}(client {{ ident (nprpcIdent "RPCClient") }}, svcSpec *{{ .GoName }}Spec) {{ .GoName }} {
	return &client{{ .GoName }}{
		{{- range .Methods }}
			handler{{ .GoName }}: client.MakeHandler(svcSpec.spec{{ .GoName }}),
		{{- end }}
	}
}

{{- $service := . -}}

{{- range .Methods }}
	func (svc *client{{ $service.GoName }}) {{ .GoName }}(ctx {{ ident (contextIdent "Context") }}, input *{{ ident .Input.GoIdent }}) (*{{ ident .Output.GoIdent }}, error) {
		output, err := svc.handler{{ .GoName }}(ctx, input)
		if err != nil {
			return nil, err
		}
		return output.(*{{ ident .Output.GoIdent }}), nil
	}
{{ end }}
`)
	if err != nil {
		log.Fatal(err)
	}

	if err := rpcStubTpl.Execute(g, service); err != nil {
		log.Fatal(err)
	}
}

func genMsgStub(g *protogen.GeneratedFile, f *protogen.File, message *protogen.Message) {

	funcs := map[string]interface{}{
		"ident":        g.QualifiedGoIdent,
		"reflectIdent": reflectIdent,
		"contextIdent": contextIdent,
		"npmsgIdent":   npmsgIdent,
	}

	msgStubTpl, err := template.New("msg").Funcs(funcs).Parse(`
{{- $goName := ident .GoIdent -}}
// --- Msg stub for {{ $goName }} ---

// {{ $goName }}Spec implements {{ ident (npmsgIdent "MsgSpec") }} interface.
type {{ $goName }}Spec string

// SubjectName returns string(spec).
func (spec {{ $goName }}Spec) SubjectName() string {
	return string(spec)
}

// NewMsg returns new({{ $goName }}).
func (spec {{ $goName }}Spec) NewMsg() interface{} {
	return new({{ $goName }})
}

var (
	msgType_{{ $goName }} = {{ ident (reflectIdent "TypeOf") }}((*{{ $goName }})(nil))
)
// MsgType returns {{ ident (reflectIdent "TypeOf") }}((*{{ $goName }})(nil)).
func (spec {{ $goName }}Spec) MsgType() {{ ident (reflectIdent "Type") }} {
	return msgType_{{ $goName }}
}

var (
	msgValue_{{ $goName }} = &{{ $goName }}{}
)
// MsgValue returns a sample *{{ $goName }}.
func (spec {{ $goName }}Spec) MsgValue() interface{} {
	return msgValue_{{ $goName }}
}

// Subscribe{{ $goName }} subscribes to a {{ $goName }} topic.
func Subscribe{{ $goName }}(
	subscriber {{ ident (npmsgIdent "MsgSubscriber") }},
	spec {{ $goName }}Spec,
	queue string,
	handler func({{ ident (contextIdent "Context") }}, *{{ $goName }}) error,
	opts ...interface{}) error {

	return subscriber.Subscribe(spec, queue, func(ctx {{ ident (contextIdent "Context") }}, msg interface{}) error {
    return handler(ctx, msg.(*{{ $goName }}))
  }, opts...)
}

// Publish{{ $goName }} publish a {{ $goName }} message.
func Publish{{ $goName }}(
	publisher {{ ident (npmsgIdent "MsgPublisher") }},
	ctx {{ ident (contextIdent "Context") }},
	spec {{ $goName }}Spec,
	msg *{{ $goName }},
) error {
	return publisher.Publish(ctx, spec, msg)
}
`)
	if err != nil {
		log.Fatal(err)
	}

	if err := msgStubTpl.Execute(g, message); err != nil {
		log.Fatal(err)
	}

}

func contextIdent(goName string) protogen.GoIdent {
	return protogen.GoIdent{
		GoName:       goName,
		GoImportPath: "context",
	}
}

func reflectIdent(goName string) protogen.GoIdent {
	return protogen.GoIdent{
		GoName:       goName,
		GoImportPath: "reflect",
	}
}

func nprpcIdent(goName string) protogen.GoIdent {
	return protogen.GoIdent{
		GoName:       goName,
		GoImportPath: "github.com/huangjunwen/nproto/v2/rpc",
	}
}

func npmsgIdent(goName string) protogen.GoIdent {
	return protogen.GoIdent{
		GoName:       goName,
		GoImportPath: "github.com/huangjunwen/nproto/v2/msg",
	}
}
