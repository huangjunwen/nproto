package nppbmd

import (
	npmd "github.com/huangjunwen/nproto/v2/md"
)

// MetaData is another npmd.MD implementation.
type MetaData map[string]*MetaDataValueList

// NewMetaData converts npmd.MD to MetaData.
func NewMetaData(md npmd.MD) MetaData {
	if md == nil {
		return nil
	}
	ret := MetaData{}
	md.Keys(func(key string) bool {
		ret[key] = &MetaDataValueList{
			Values: md.Values(key),
		}
		return true
	})
	return ret
}

// Keys implements npmd.MD interface.
func (md MetaData) Keys(cb func(string) bool) {
	if md == nil {
		return
	}
	for key := range md {
		if ok := cb(key); !ok {
			return
		}
	}
	return
}

// HasKey implements npmd.MD interface.
func (md MetaData) HasKey(key string) bool {
	_, ok := md[key]
	return ok
}

// Values implements npmd.MD interface.
func (md MetaData) Values(key string) [][]byte {
	v := md[key]
	if v == nil {
		return nil
	}
	return v.Values
}

var (
	_ npmd.MD = (MetaData)(nil)
)
