package nppbmd

import (
	npmd "github.com/huangjunwen/nproto/v2/md"
)

// To converts to npmd.MetaData.
func (md *MD) To() npmd.MetaData {
	ret := npmd.MetaData{}
	for k, v := range md.KeyValues {
		ret[k] = v.Values
	}
	return ret
}

// NewMD converts from npmd.MD.
func NewMD(src npmd.MD) *MD {
	md := &MD{KeyValues: make(map[string]*ValueList)}
	src.Keys(func(k string) bool {
		md.KeyValues[k] = &ValueList{Values: src.Values(k)}
		return true
	})
	return md
}
