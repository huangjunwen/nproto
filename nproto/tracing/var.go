package tracing

import (
	"fmt"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

var (
	// ComponentTag is attached to each sapn.
	ComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "nproto.tracing",
	}
	// TracingMDKey is the nproto.MD key to carry tracing data.
	TracingMDKey = "NProtoTracing"
	// ClientHandlerOpNameFmt is used to generate operation name of a RPC client handler span.
	ClientHandlerOpNameFmt = func(svcName string, method *nproto.RPCMethod) string {
		return fmt.Sprintf("RPC Client %s:%s", svcName, method.Name)
	}
	// ServerHandlerOpNameFmt is used to generate operation name of a RPC server handler span.
	ServerHandlerOpNameFmt = func(svcName string, method *nproto.RPCMethod) string {
		return fmt.Sprintf("RPC Server %s:%s", svcName, method.Name)
	}
	// PublishOpNameFmt is used to generate operation name of a msg publish span.
	PublishOpNameFmt = func(subject string) string {
		return fmt.Sprintf("Msg Publisher %s", subject)
	}
	// PublishAsyncOpNameFmt is used to generate operation name of a async msg publish span.
	PublishAsyncOpNameFmt = func(subject string) string {
		return fmt.Sprintf("Msg PublishAsync %s", subject)
	}
	// SubscriberHandlerOpNameFmt is used to generate operation name of a subscription handler span.
	SubscriberHandlerOpNameFmt = func(subject, queue string) string {
		return fmt.Sprintf("Msg Subscriber %s:%s", subject, queue)
	}
)
