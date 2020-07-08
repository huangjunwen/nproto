package natsrpc

import (
	"sync/atomic"

	. "github.com/huangjunwen/nproto/v2/enc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

// methodMap stores method info for a service.
type methodMap struct {
	// map[methodName]*methodInfo
	v atomic.Value
}

type methodInfo struct {
	Spec     *RPCSpec
	Handler  RPCHandler
	Encoders map[string]Encoder // supported encoders for this method
}

func newMethodMap() *methodMap {
	ret := &methodMap{}
	ret.v.Store(make(map[string]*methodInfo))
	return ret
}

func (mm *methodMap) Get() map[string]*methodInfo {
	return mm.v.Load().(map[string]*methodInfo)
}

// NOTE: though the update is atomic, but this function should be wrapped by a mutex to serialize updates.
func (mm *methodMap) RegistHandler(spec *RPCSpec, handler RPCHandler, encoders map[string]Encoder) {

	// Create a deep copy instead.
	m := mm.Get()
	newM := make(map[string]*methodInfo)
	for name, info := range m {
		newM[name] = info
	}

	// Add/Del.
	if handler != nil {
		newM[spec.MethodName] = &methodInfo{
			Spec:     spec,
			Handler:  handler,
			Encoders: encoders,
		}
	} else {
		delete(newM, spec.MethodName)
	}

	// Atomic replace.
	mm.v.Store(newM)
}
