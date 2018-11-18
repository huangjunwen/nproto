package dbstore

var (
	// NOTE: Possible to use a sync.Pool if needed.
	newNode    = func() *msgNode { return &msgNode{} }
	deleteNode = func(node *msgNode) {}
)

// msgList is a list of messages.
type msgList struct {
	head msgNode // head.next is the first item and head.prev is the last item.
}

// msgNode is a single message. It belong to at most one msgList at any time.
type msgNode struct {
	Id      uint64
	Subject string
	Data    []byte

	prev *msgNode
	next *msgNode
}

// newMsgList creates an empty msgList.
func newMsgList() *msgList {
	ret := &msgList{}
	head := &ret.head
	head.prev = head
	head.next = head
	return ret
}

// NewNode creates a new msgNode and appends it to list.
func (list *msgList) NewNode() *msgNode {
	ret := newNode()
	ret.attach(list)
	return ret
}

// AppendNode detaches a msgNode from its current owning list and appends to list.
func (list *msgList) AppendNode(node *msgNode) {
	node.detach()
	node.attach(list)
}

// Reset deletes all msgNode in a msgList.
func (list *msgList) Reset() {
	head := &list.head
	node := head.next
	for node != head {
		next := node.next
		deleteNode(node)
		node = next
	}
	head.next = head
	head.prev = head
}

// MigrateTo migrates all msgNode from list to target.
func (list *msgList) MigrateTo(target *msgList) {
	head := &list.head
	if head.next == head {
		return
	}
	first := head.next
	last := head.prev
	first.prev = &target.head
	last.next = &target.head
	target.head.next = first
	target.head.prev = last

	head.next = head
	head.prev = head
}

func (node *msgNode) attach(list *msgList) {
	head := &list.head
	node.prev = head.prev
	node.next = head
	head.prev.next = node
	head.prev = node
}

func (node *msgNode) detach() {
	prev := node.prev
	next := node.next
	if prev != nil {
		prev.next = next
	}
	if next != nil {
		next.prev = prev
	}
	node.prev = nil
	node.next = nil
}
