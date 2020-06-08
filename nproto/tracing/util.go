package tracing

import (
	"bytes"
	"context"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

// spanCtxFromCtx gets the span context from context. Returns nil if there is no span set.
func spanCtxFromCtx(ctx context.Context) ot.SpanContext {
	if span := ot.SpanFromContext(ctx); span != nil {
		return span.Context()
	}
	return nil
}

// injectSpanCtx adds span context to `md` and creates a new nproto.MD. `md` can be nil.
func injectSpanCtx(tracer ot.Tracer, spanCtx ot.SpanContext, md nproto.MD) (nproto.MD, error) {
	if md == nil {
		md = nproto.EmptyMD
	}
	w := &bytes.Buffer{}
	if err := tracer.Inject(spanCtx, ot.Binary, w); err != nil {
		return nil, err
	}
	return newWithTracingMD(md, w.Bytes()), nil
}

// extractSpanCtx extracts span context from `md` or nil if not found. `md` can be nil.
func extractSpanCtx(tracer ot.Tracer, md nproto.MD) (ot.SpanContext, error) {
	if md == nil {
		return nil, nil
	}
	data := md.Values(TracingMDKey)
	if len(data) == 0 {
		return nil, nil
	}
	r := bytes.NewReader(data[0])
	spanCtx, err := tracer.Extract(ot.Binary, r)
	if err == ot.ErrSpanContextNotFound {
		err = nil
	}
	return spanCtx, err
}

// setSpanError set error on the span.
func setSpanError(span ot.Span, err error) {
	if err != nil {
		otext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
	}
}
