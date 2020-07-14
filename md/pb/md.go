package pb

import (
	npmd "github.com/huangjunwen/nproto/v2/md"
)

// To writes key/values to npmd.MetaData.
func (md *MD) To() npmd.MetaData {
	ret := npmd.MetaData{}
	for k, v := range md.KeyValues {
		ret[k] = v.Values
	}
	return ret
}

// From reads key/values from src.
func (md *MD) From(src npmd.MD) error {
	md.KeyValues = make(map[string]*ValueList)
	return src.Keys(func(k string) error {
		md.KeyValues[k] = &ValueList{
			Values: src.Values(k),
		}
		return nil
	})
}
