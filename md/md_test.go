package md

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
