package md

import (
	"fmt"
)

// MetaData is the default implementation of MD. nil is a valid value. See EmptyMD.
type MetaData map[string][][]byte

// NewMetaDataPairs creates a MetaData from key/value pairs. len(kv) must be even.
func NewMetaDataPairs(kv ...string) MetaData {
	if len(kv)%2 == 1 {
		panic(fmt.Errorf("NewMetaDataPairs: got odd number of kv strings"))
	}
	md := MetaData{}
	for i := 0; i < len(kv); i += 2 {
		md[kv[i]] = [][]byte{[]byte(kv[i+1])}
	}
	return md
}

// NewMetaDataFromMD creates a MetaData from MD.
func NewMetaDataFromMD(md MD) MetaData {
	if md == nil {
		return (MetaData)(nil)
	}
	ret := MetaData{}
	md.Keys(func(key string) bool {
		// XXX: Copy?
		ret[key] = md.Values(key)
		return true
	})
	return ret
}

// Keys implements MD interface.
func (md MetaData) Keys(cb func(string) bool) {
	if len(md) == 0 {
		return
	}
	for key, _ := range md {
		if ok := cb(key); !ok {
			return
		}
	}
	return
}

// HasKey implements MD interface.
func (md MetaData) HasKey(key string) bool {
	_, ok := md[key]
	return ok
}

// Values implements MD interface.
func (md MetaData) Values(key string) [][]byte {
	return md[key]
}

var (
	// EmptyMD is an empty MD.
	EmptyMD MD = (MetaData)(nil)
)
