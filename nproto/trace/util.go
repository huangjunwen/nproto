package trace

import (
	"context"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/huangjunwen/nproto/nproto"
)

type mdReaderWriter struct {
	md nproto.MetaData
}

func (rw mdReaderWriter) Set(key, val string) {
	rw.md.Append(key, val)
}

func (rw mdReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for key, vals := range rw.md {
		for _, val := range vals {
			if err := handler(key, val); err != nil {
				return err
			}
		}
	}
	return nil
}

// spanCtxFromCtx gets the span context from context. Returns nil if there is no span set.
func spanCtxFromCtx(ctx context.Context) ot.SpanContext {
	if span := ot.SpanFromContext(ctx); span != nil {
		return span.Context()
	}
	return nil
}

// injectSpanCtx injects span context into outgoing MetaData. md can be nil.
// Returns the new MetaData (copy).
func injectSpanCtx(tracer ot.Tracer, spanCtx ot.SpanContext, md nproto.MetaData) (nproto.MetaData, error) {
	md = md.Copy()
	if err := tracer.Inject(spanCtx, ot.TextMap, mdReaderWriter{md}); err != nil {
		return nil, err
	}
	return md, nil
}

// extractSpanCtx extracts span context from incoming MetaData. md can be nil.
// Returns the span context if found or nil otherwise.
func extractSpanCtx(tracer ot.Tracer, md nproto.MetaData) (ot.SpanContext, error) {
	if len(md) == 0 {
		return nil, nil
	}
	spanCtx, err := tracer.Extract(ot.TextMap, mdReaderWriter{md})
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
