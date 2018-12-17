package trace

import (
	opentracing "github.com/opentracing/opentracing-go"
	opentracingext "github.com/opentracing/opentracing-go/ext"

	"github.com/huangjunwen/nproto/nproto"
)

var (
	ComponentTag = opentracing.Tag{
		Key:   string(opentracingext.Component),
		Value: "nproto",
	}
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
