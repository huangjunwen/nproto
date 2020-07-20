package nppbmd

import (
	npmd "github.com/huangjunwen/nproto/v2/md"
)

// MetaData is another npmd.MD implementation.
type MetaData map[string]*MetaDataValueList

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

func (md MetaData) Keys(cb func(string) bool) {
	if md == nil {
		return
	}
	for key, _ := range md {
		if ok := cb(key); !ok {
			return
		}
	}
	return
}

func (md MetaData) HasKey(key string) bool {
	_, ok := md[key]
	return ok
}

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
