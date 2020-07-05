package msg

import (
	"fmt"
	"reflect"
)

// MsgSpec is the contract between msg publisher and subscriber.
//
// It should be filled and call Validate() before use. After that don't modify its content.
type MsgSpec struct {
	// SubjectName is the topic.
	SubjectName string

	// NewMsg is used to generate a new message. Must be a pointer.
	NewMsg func() interface{}

	msgType reflect.Type
}

// Validate the MsgSpec. Call this before any other methods.
func (spec *MsgSpec) Validate() error {
	if spec.Validated() {
		return nil
	}

	if spec.SubjectName == "" {
		return fmt.Errorf("MsgSpec.SubjectName is empty")
	}
	if spec.NewMsg == nil {
		return fmt.Errorf("MsgSpec.NewMsg is empty")
	}

	newMsg := spec.NewMsg()
	if newMsg == nil {
		return fmt.Errorf("MsgSpec.NewMsg() returns nil")
	}
	msgType := reflect.TypeOf(newMsg)
	if msgType.Kind() != reflect.Ptr {
		return fmt.Errorf("MsgSpec.NewMsg() returns non-pointer")
	}

	spec.msgType = msgType
	return nil
}

func (spec *MsgSpec) Validated() bool {
	return spec.msgType != nil
}

// AssertMsgType makes sure msg's type conform to the spec.
// It will panic if NewMsg() returns nil or non-pointer, or msg's type is different.
func (spec *MsgSpec) AssertMsgType(msg interface{}) error {
	if !spec.Validated() {
		return fmt.Errorf("MsgSpec has not validated yet")
	}
	if msgType := reflect.TypeOf(msg); msgType != spec.msgType {
		return fmt.Errorf("%s msg expect %s, but got %s", spec.String(), spec.msgType.String(), msgType.String())
	}
	return nil
}

func (spec *MsgSpec) String() string {
	if !spec.Validated() {
		return fmt.Sprintf("MsgSpec(%s)", spec.SubjectName)
	}
	return fmt.Sprintf("MsgSpec(%s %s)", spec.SubjectName, spec.msgType.String())
}
