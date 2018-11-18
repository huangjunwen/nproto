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
	deleteNodeC := make(chan uint64, 10000)
	deleteNode = func(node *msgNode) {
		deleteNodeC <- node.Id
		origDeleteNode(node)
	}
	expectDeleteNode := func(expectId uint64) {
		id := <-deleteNodeC
		assert.Equal(id, expectId)
	}

	// --- test NewNode ---
	l1 := newMsgList()
	{
		assert.Equal(&l1.head, l1.head.prev)
		assert.Equal(&l1.head, l1.head.next)
	}

	n1 := l1.NewNode()
	n1.Id = 1
	{
		assert.Equal(n1, l1.head.next) // First item is n1.
		assert.Equal(n1, l1.head.prev) // Last item is n1.
		assert.Equal(&l1.head, n1.prev)
		assert.Equal(&l1.head, n1.next)
	}
	n2 := l1.NewNode()
	n2.Id = 2
	{
		assert.Equal(n1, l1.head.next) // First item is n1.
		assert.Equal(n2, l1.head.prev) // Last item is n2.
		assert.Equal(&l1.head, n1.prev)
		assert.Equal(n2, n1.next)
		assert.Equal(n1, n2.prev)
		assert.Equal(&l1.head, n2.next)
	}

	// --- test AppendNode ---
	l2 := newMsgList()

	l2.AppendNode(n1)
	{
		assert.Equal(n2, l1.head.next) // First item is n2.
		assert.Equal(n2, l1.head.prev) // Last item is n2.
		assert.Equal(&l1.head, n2.prev)
		assert.Equal(&l1.head, n2.next)
		assert.Equal(n1, l2.head.next) // First item is n1.
		assert.Equal(n1, l2.head.prev) // Last item is n1.
		assert.Equal(&l2.head, n1.prev)
		assert.Equal(&l2.head, n1.next)
	}

	l2.AppendNode(n2)
	{
		assert.Equal(&l1.head, l1.head.next) // l1 is empty now.
		assert.Equal(&l1.head, l1.head.prev) // l1 is empty now.
		assert.Equal(n1, l2.head.next)       // First item is n1.
		assert.Equal(n2, l2.head.prev)       // Last item is n2.
		assert.Equal(&l2.head, n1.prev)
		assert.Equal(n2, n1.next)
		assert.Equal(n1, n2.prev)
		assert.Equal(&l2.head, n2.next)
	}

	// --- test Iterate ---
	{
		iter := l2.Iterate()
		assert.Equal(n1, iter())
		assert.Equal(n2, iter())
		assert.Equal((*msgNode)(nil), iter())
	}
	{
		iter := l1.Iterate()
		assert.Equal((*msgNode)(nil), iter())
	}

	// --- test Reset ---
	l2.Reset()
	{
		expectDeleteNode(1)
		expectDeleteNode(2)
		assert.Len(deleteNodeC, 0)
		assert.Equal(&l2.head, l2.head.next) // l2 is empty now.
		assert.Equal(&l2.head, l2.head.prev) // l2 is empty now.
	}

}
