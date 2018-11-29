package dbstore

// msgNode is a single message.
type msgNode struct {
	Id      int64
	Subject string
	Data    []byte

	list *msgList
	next *msgNode
}

// msgList is a list of messages.
type msgList struct {
	head *msgNode
	tail *msgNode
	n    int
}

// msgStream is a stream (unknown length) of messages.
// The next message (or nil if no more message) should be returned when calling with true.
// If calling with false, then the stream should be closed and release resource.
type msgStream func(next bool) (*msgNode, error)

// Append appends node to list. node must not belong to any list.
func (list *msgList) Append(node *msgNode) {
	if node.list != nil {
		panic("msgNode already belongs to some msgList")
	}
	node.list = list

	if list.n == 0 {
		list.head = node
		list.tail = node
	} else {
		list.tail.next = node
		list.tail = node
	}
	list.n += 1
}

// Pop pops the first node from list. It returns nil if the list is empty.
func (list *msgList) Pop() *msgNode {
	node := list.head
	if node == nil {
		return nil
	}

	list.n -= 1
	if list.n == 0 {
		list.head = nil
		list.tail = nil
	} else {
		list.head = node.next
	}

	node.list = nil
	node.next = nil
	return node
}

// Reset deletes all msgNode in a msgList.
func (list *msgList) Reset() {
	list.head = nil
	list.tail = nil
	list.n = 0
}

// Iterate returns an iterator of the list.
func (list *msgList) Iterate() func() *msgNode {
	node := list.head
	return func() *msgNode {
		if node == nil {
			return nil
		}
		ret := node
		node = node.next
		return ret
	}
}

// newErrStream creates a msgStream that always returns an error.
func newErrStream(err error) msgStream {
	return func(next bool) (*msgNode, error) {
		return nil, err
	}
}
