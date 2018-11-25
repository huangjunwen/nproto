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
	Id      int64
	Subject string
	Data    []byte

	list *msgList
	next *msgNode
}

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
