package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaData(t *testing.T) {
	assert := assert.New(t)

	{
		assert.NoError(EmptyMD.Keys(func(_ string) error { return nil }))
		assert.False(EmptyMD.HasKey("hello"))
		assert.Nil(EmptyMD.Values("hello"))
	}

	{
		assert.Panics(func() {
			NewMetaDataPairs("1", "2", "3")
		})

		md := NewMetaDataPairs("1", "2")
		assert.NoError(md.Keys(func(key string) error {
			assert.Equal("1", key)
			return nil
		}))
		assert.True(md.HasKey("1"))
		assert.Equal([][]byte{[]byte("2")}, md.Values("1"))
	}

	{
		assert.Equal(EmptyMD, NewMetaDataFromMD(nil))
	}

}
