package tracing

import (
	"fmt"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

var (
	ComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "nproto.tracing",
	}
	TracingMDKey           = "NProtoTracing"
	ClientHandlerOpNameFmt = func(svcName string, method *nproto.RPCMethod) string {
		return fmt.Sprintf("ClientHandler::%s::%s", svcName, method.Name)
	}
	ServerHandlerOpNameFmt = func(svcName string, method *nproto.RPCMethod) string {
		return fmt.Sprintf("ServerHandler::%s::%s", svcName, method.Name)
	}
	PublishOpNameFmt = func(subject string) string {
		return fmt.Sprintf("Publish::%s", subject)
	}
	PublishAsyncOpNameFmt = func(subject string) string {
		return fmt.Sprintf("PublishAsync::%s", subject)
	}
	SubscriberHandlerOpNameFmt = func(subject, queue string) string {
		return fmt.Sprintf("SubscriberHandler::%s::%s", subject, queue)
	}
)
