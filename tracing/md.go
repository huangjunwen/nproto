package tracing

import (
	npmd "github.com/huangjunwen/nproto/v2/md"
)

type tracingMD struct {
	md   npmd.MD
	data [1][]byte
}

var (
	// TracingMDKey is the meta data key to carry tracing data.
	TracingMDKey = "nptracing"
)

var (
	_ npmd.MD = (*tracingMD)(nil)
)

func newTracingMD(md npmd.MD, data []byte) *tracingMD {
	ret := &tracingMD{
		md: md,
	}
	ret.data[0] = data
	return ret
}

func (md *tracingMD) Keys(cb func(string) bool) {
	if ok := cb(TracingMDKey); !ok {
		return
	}

	md.md.Keys(func(key string) bool {
		// Skip TracingMDKey in origin meta data.
		if key == TracingMDKey {
			return true
		}
		return cb(key)
	})
}

func (md *tracingMD) HasKey(key string) bool {
	switch key {
	case TracingMDKey:
		return true
	default:
		return md.md.HasKey(key)
	}
}

func (md *tracingMD) Values(key string) [][]byte {
	switch key {
	case TracingMDKey:
		return md.data[:]
	default:
		return md.md.Values(key)
	}
}
