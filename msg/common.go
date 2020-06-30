package msg

// MsgSpec is the contract between msg publisher and subscriber.
type MsgSpec struct {
	// SubjectName is the topic.
	SubjectName string

	// NewMsg is used to generate a new message. Should be a pointer.
	NewMsg func() interface{}
}
