package msg

import (
	"fmt"
	"reflect"
)

// MsgSpec is the contract between msg publisher and subscriber.
// It should be readonly. Don't change its content once it is filled.
type MsgSpec struct {
	// SubjectName is the topic.
	SubjectName string

	// NewMsg is used to generate a new message. Must be a pointer.
	NewMsg func() interface{}

	msgType reflect.Type
}

// AssertMsgType makes sure msg's type conform to the spec.
// It will panic if NewMsg() returns nil or non-pointer, or msg's type is different.
func (spec *MsgSpec) AssertMsgType(msg interface{}) {
	if spec.msgType == nil {
		newMsg := spec.NewMsg()
		if newMsg == nil {
			panic(fmt.Errorf("%s NewMsg() returns nil", spec.String()))
		}
		msgType := reflect.TypeOf(newMsg)
		if msgType.Kind() != reflect.Ptr {
			panic(fmt.Errorf("%s NewMsg() returns non-pointer", spec.String()))
		}
		spec.msgType = msgType
	}
	if msgType := reflect.TypeOf(msg); msgType != spec.msgType {
		panic(fmt.Errorf("%s msg expect %s, but got %s", spec.String(), spec.msgType.String(), msgType.String()))
	}
}

func (spec *MsgSpec) String() string {
	return fmt.Sprintf("MsgSpec(%s)", spec.SubjectName)
}
