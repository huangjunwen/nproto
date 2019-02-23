package tracing

import (
	"context"
	"strings"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/huangjunwen/nproto/nproto"
)

// spanCtxFromCtx gets the span context from context. Returns nil if there is no span set.
func spanCtxFromCtx(ctx context.Context) ot.SpanContext {
	if span := ot.SpanFromContext(ctx); span != nil {
		return span.Context()
	}
	return nil
}

// injectSpanCtx injects span context into `md[key]`. Old value is overwritten.
func injectSpanCtx(tracer ot.Tracer, spanCtx ot.SpanContext, md nproto.MetaData, key string) error {
	w := &strings.Builder{}
	if err := tracer.Inject(spanCtx, ot.Binary, w); err != nil {
		return err
	}
	md.Set(key, w.String())
	return nil
}

// extractSpanCtx extracts span context from `md[key]`. nil is returned if not found.
func extractSpanCtx(tracer ot.Tracer, md nproto.MetaData, key string) (ot.SpanContext, error) {
	if len(md) == 0 {
		return nil, nil
	}
	r := strings.NewReader(md.Get(key))
	spanCtx, err := tracer.Extract(ot.Binary, r)
	if err == ot.ErrSpanContextNotFound {
		// ErrSpanContextNotFound is not error.
		err = nil
	}
	return spanCtx, err
}

// setSpanError set error on the span.
func setSpanError(span ot.Span, err error) {
	if err != nil {
		otext.Error.Set(span, true)
		span.LogFields(otlog.String("event", "error"), otlog.String("message", err.Error()))
	}
}
