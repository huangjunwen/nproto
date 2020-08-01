package msgtracing

import (
	"fmt"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	. "github.com/huangjunwen/nproto/v2/msg"
)

var (
	// PublisherComponentTag is added to each publisher span.
	PublisherComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "npmsg.publisher",
	}
	// AsyncPublisherComponentTag is added to each async publisher span.
	AsyncPublisherComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "npmsg.async.publisher",
	}
	// SubscriberComponentTag is added to each subscriber span.
	SubscriberComponentTag = ot.Tag{
		Key:   string(otext.Component),
		Value: "npmsg.subscriber",
	}
)

var (
	// PublisherOpName is used to generate operation name of a msg publish span.
	PublisherOpName = func(spec MsgSpec) string {
		return fmt.Sprintf("Msg Publisher %s", spec.SubjectName())
	}
	// AsyncPublisherOpName is used to generate operation name of a async msg publish span.
	AsyncPublisherOpName = func(spec MsgSpec) string {
		return fmt.Sprintf("Msg AsyncPublisher %s", spec.SubjectName())
	}
	// SubscriberOpName is used to generate operation name of a subscriber span.
	SubscriberOpName = func(spec MsgSpec, queue string) string {
		return fmt.Sprintf("Msg Subscriber %s:%s", spec.SubjectName(), queue)
	}
)
