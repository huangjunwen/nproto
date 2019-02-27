package nproto

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMD(t *testing.T) {
	assert := assert.New(t)

	assert.Nil(MDFromOutgoingContext(context.Background()))
	assert.Nil(MDFromIncomingContext(context.Background()))
	{
		md := MetaData{}
		ctx := NewOutgoingContextWithMD(context.Background(), md)
		assert.Equal(md, MDFromOutgoingContext(ctx))
		assert.Nil(MDFromIncomingContext(ctx))
	}
	{
		md := MetaData{}
		ctx := NewIncomingContextWithMD(context.Background(), md)
		assert.Equal(md, MDFromIncomingContext(ctx))
		assert.Nil(MDFromOutgoingContext(ctx))
	}
}

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
