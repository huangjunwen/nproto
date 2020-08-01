package tracing

import (
	"bytes"
	"context"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"

	npmd "github.com/huangjunwen/nproto/v2/md"
)

// SpanCtxFromCtx gets span context from context. Returns nil if there is no span set.
func SpanCtxFromCtx(ctx context.Context) ot.SpanContext {
	if span := ot.SpanFromContext(ctx); span != nil {
		return span.Context()
	}
	return nil
}

// InjectSpanCtx adds span context to `md` (a new one is created). `md` can be nil.
func InjectSpanCtx(tracer ot.Tracer, spanCtx ot.SpanContext, md npmd.MD) (npmd.MD, error) {
	w := &bytes.Buffer{}
	if err := tracer.Inject(spanCtx, ot.Binary, w); err != nil {
		return nil, err
	}
	if md == nil {
		md = npmd.EmptyMD
	}
	return newTracingMD(md, w.Bytes()), nil
}

// ExtractSpanCtx extracts span context from `md` or nil if not found.
func ExtractSpanCtx(tracer ot.Tracer, md npmd.MD) (ot.SpanContext, error) {
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

// SetSpanError set error on the span. If `err` is nil, then nop.
func SetSpanError(span ot.Span, err error) {
	if err != nil {
		otext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
	}
}
