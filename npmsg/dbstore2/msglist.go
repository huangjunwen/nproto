package dbstore

var (
	// NOTE: Possible to use a sync.Pool if needed.
	newNode    = func() *msgNode { return &msgNode{} }
	deleteNode = func(node *msgNode) {}
)

// msgList is a list of messages.
type msgList struct {
	head *msgNode
	tail *msgNode
	n    int
}

// msgNode is a single message.
type msgNode struct {
	Id      uint64
	Subject string
	Data    []byte
	Err     error

	list *msgList
	next *msgNode
}

// Append appends node to list. node must not belong to other list.
func (list *msgList) Append(node *msgNode) {
	if node.list != nil {
		if node.list != list {
			panic("msgNode belongs to other msgList")
		}
		return
	}

	node.list = list
	if list.head == nil {
		list.head = node
	}
	if list.tail != nil {
		list.tail.next = node
	}
	list.tail = node
	list.n += 1
}

// Reset deletes all msgNode in a msgList.
func (list *msgList) Reset() {
	node := list.head
	for node != nil {
		next := node.next
		node.list = nil
		node.next = nil
		deleteNode(node)
		node = next
	}
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
