package dbstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgList(t *testing.T) {
	assert := assert.New(t)

	// --- test Append ---
	l := &msgList{}
	{
		assert.Nil(l.head)
		assert.Nil(l.tail)
		assert.Equal(0, l.n)
	}

	n1 := &msgNode{
		Id: 1,
	}
	l.Append(n1)
	{
		assert.Equal(n1, l.head)
		assert.Equal(n1, l.tail)
		assert.Equal(1, l.n)
		assert.Equal(l, n1.list)
		assert.Nil(n1.next)
	}

	n2 := &msgNode{
		Id: 2,
	}
	l.Append(n2)
	{
		assert.Equal(n1, l.head)
		assert.Equal(n2, l.tail)
		assert.Equal(2, l.n)
		assert.Equal(l, n1.list)
		assert.Equal(n2, n1.next)
		assert.Equal(l, n2.list)
		assert.Nil(n2.next)
	}

	// --- test Pop ---
	{
		assert.Equal(n1, l.Pop())
		assert.Equal(n2, l.head)
		assert.Equal(n2, l.tail)
		assert.Equal(1, l.n)
		assert.Nil(n1.list)
		assert.Nil(n1.next)
	}

	{
		assert.Equal(n2, l.Pop())
		assert.Nil(l.head)
		assert.Nil(l.tail)
		assert.Equal(0, l.n)
		assert.Nil(n2.list)
		assert.Nil(n2.next)
	}

	// --- test Iterate ---
	l.Append(n1)
	l.Append(n2)
	{
		iter := l.Iterate()
		assert.Equal(n1, iter())
		assert.Equal(n2, iter())
		assert.Equal((*msgNode)(nil), iter())
	}
	{
		l2 := &msgList{}
		iter := l2.Iterate()
		assert.Equal((*msgNode)(nil), iter())
	}

	// --- test Reset ---
	l.Reset()
	{
		assert.Nil(l.head)
		assert.Nil(l.tail)
		assert.Equal(0, l.n)
	}

}
