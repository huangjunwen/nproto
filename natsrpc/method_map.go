package natsrpc

import (
	"sync"

	. "github.com/huangjunwen/nproto/v2/enc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

// methodMap stores method info for a service.
type methodMap struct {
	// XXX: sync.Map ?
	mu sync.RWMutex
	v  map[string]*methodInfo
}

type methodInfo struct {
	Spec     *RPCSpec
	Handler  RPCHandler
	Encoders map[string]Encoder // supported encoders for this method
}

func newMethodMap() *methodMap {
	return &methodMap{
		v: make(map[string]*methodInfo),
	}
}

func (mm *methodMap) Lookup(methodName string) *methodInfo {
	mm.mu.RLock()
	info := mm.v[methodName]
	mm.mu.RUnlock()
	return info
}

func (mm *methodMap) Regist(spec *RPCSpec, handler RPCHandler, encoders map[string]Encoder) {

	if handler == nil {
		mm.mu.Lock()
		delete(mm.v, spec.MethodName)
		mm.mu.Unlock()
		return
	}

	info := &methodInfo{
		Spec:     spec,
		Handler:  handler,
		Encoders: encoders,
	}
	mm.mu.Lock()
	mm.v[spec.MethodName] = info
	mm.mu.Unlock()
	return

}
