package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaData(t *testing.T) {
	assert := assert.New(t)

	{
		assert.False(EmptyMD.HasKey("hello"))
		assert.Nil(EmptyMD.Values("hello"))
	}

	{
		assert.Panics(func() {
			NewMetaDataPairs("1", "2", "3")
		})

		md := NewMetaDataPairs("1", "2")
		md.Keys(func(key string) bool {
			assert.Equal("1", key)
			return true
		})
		assert.True(md.HasKey("1"))
		assert.Equal([][]byte{[]byte("2")}, md.Values("1"))
	}

	{
		assert.Equal(EmptyMD, NewMetaDataFromMD(nil))
	}

}
