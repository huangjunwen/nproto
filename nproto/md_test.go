package nproto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaData(t *testing.T) {
	assert := assert.New(t)

	{
		md := (MetaData)(nil)
		md2 := md.Copy()
		assert.NotNil(md2)
	}

}
