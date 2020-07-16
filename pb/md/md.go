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
func NewMD(src npmd.MD) (*MD, error) {
	md := &MD{KeyValues: make(map[string]*ValueList)}
	if err := src.Keys(func(k string) error {
		md.KeyValues[k] = &ValueList{Values: src.Values(k)}
		return nil
	}); err != nil {
		return nil, err
	}
	return md, nil
}

// MustMD is must version of NewMD.
func MustMD(src npmd.MD) *MD {
	md, err := NewMD(src)
	if err != nil {
		panic(err)
	}
	return md
}
