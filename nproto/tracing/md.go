package tracing

import (
	"github.com/huangjunwen/nproto/nproto"
)

type withTracingMD struct {
	md   nproto.MD
	data [1][]byte
}

var (
	_ nproto.MD = (*withTracingMD)(nil)
)

func newWithTracingMD(md nproto.MD, data []byte) *withTracingMD {
	ret := &withTracingMD{
		md: md,
	}
	ret.data[0] = data
	return ret
}

func (md *withTracingMD) Keys(cb func(string) error) error {
	// Iterate all keys in origin MD.
	if err := md.md.Keys(cb); err != nil {
		return err
	}

	// If origin MD hasn't TracingMDKey, then add it.
	if !md.md.HasKey(TracingMDKey) {
		return cb(TracingMDKey)
	}
	return nil
}

func (md *withTracingMD) HasKey(key string) bool {
	switch key {
	case TracingMDKey:
		return true
	default:
		return md.md.HasKey(key)
	}
}

func (md *withTracingMD) Values(key string) [][]byte {
	switch key {
	case TracingMDKey:
		return md.data[:]
	default:
		return md.md.Values(key)
	}
}
