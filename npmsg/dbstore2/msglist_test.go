package dbstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgList(t *testing.T) {
	assert := assert.New(t)
	origDeleteNode := deleteNode
	defer func() {
		deleteNode = origDeleteNode
	}()
	deleteNodeC := make(chan int64, 10000)
	deleteNode = func(node *msgNode) {
		deleteNodeC <- node.Id
		origDeleteNode(node)
	}
	expectDeleteNode := func(expectId int64) {
		id := <-deleteNodeC
		assert.Equal(id, expectId)
	}

	// --- test Append ---
	l := &msgList{}
	{
		assert.Nil(l.head)
		assert.Nil(l.tail)
		assert.Equal(0, l.n)
	}

	n1 := newNode()
	n1.Id = 1
	l.Append(n1)
	{
		assert.Equal(n1, l.head)
		assert.Equal(n1, l.tail)
		assert.Equal(1, l.n)
		assert.Equal(l, n1.list)
		assert.Nil(n1.next)
	}

	n2 := newNode()
	n2.Id = 2
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

	// --- test Iterate ---
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
		expectDeleteNode(1)
		expectDeleteNode(2)
		assert.Len(deleteNodeC, 0)
		assert.Nil(l.head)
		assert.Nil(l.tail)
		assert.Equal(0, l.n)
	}

}
