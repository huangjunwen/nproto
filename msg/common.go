// Package msg contains high level types/interfaces for msg implementations.
package msg

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/huangjunwen/nproto/v2/enc/rawenc"
)

var (
	// SubjectNameRegexp is subject name's format.
	SubjectNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9-_]+(\.[a-zA-Z0-9-_]+)*$`)
)

// MsgSpec is the contract between msg publisher and subscriber.
type MsgSpec interface {
	// SubjectName is the topic.
	SubjectName() string

	// NewMsg generate a new message. Must be a pointer.
	NewMsg() interface{}

	/*
		The followings are derived from NewMsg, mainly for convenience of type assertions
		and type checks.
	*/

	// MsgType returns msg's type.
	MsgType() reflect.Type

	// MsgValue returns a sample msg value, don't modify its content.
	MsgValue() interface{}
}

type msgSpec struct {
	subjectName string
	newMsg      func() interface{}
	msgType     reflect.Type
	msgValue    interface{}
}

type rawDataMsgSpec string

// MustMsgSpec is must-version of NewMsgSpec.
func MustMsgSpec(subjectName string, newMsg func() interface{}) MsgSpec {
	spec, err := NewMsgSpec(subjectName, newMsg)
	if err != nil {
		panic(err)
	}
	return spec
}

// NewMsgSpec validates and creates a new MsgSpec.
func NewMsgSpec(subjectName string, newMsg func() interface{}) (MsgSpec, error) {
	if !SubjectNameRegexp.MatchString(subjectName) {
		return nil, fmt.Errorf("SubjectName format invalid")
	}

	if newMsg == nil {
		return nil, fmt.Errorf("NewMsg is empty")
	}
	msgValue := newMsg()
	if msgValue == nil {
		return nil, fmt.Errorf("NewMsg() returns nil")
	}
	msgType := reflect.TypeOf(msgValue)
	if msgType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("NewMsg() returns %s which is not a pointer", msgType.String())
	}

	return &msgSpec{
		subjectName: subjectName,
		newMsg:      newMsg,
		msgType:     msgType,
		msgValue:    msgValue,
	}, nil
}

func (spec *msgSpec) SubjectName() string {
	return spec.subjectName
}

func (spec *msgSpec) NewMsg() interface{} {
	return spec.newMsg()
}

func (spec *msgSpec) MsgType() reflect.Type {
	return spec.msgType
}

func (spec *msgSpec) MsgValue() interface{} {
	return spec.msgValue
}

func (spec *msgSpec) String() string {
	return fmt.Sprintf("MsgSpec(%s %s)", spec.subjectName, spec.msgType.String())
}

func MustRawDataMsgSpec(subjectName string) MsgSpec {
	spec, err := NewRawDataMsgSpec(subjectName)
	if err != nil {
		panic(err)
	}
	return spec
}

func NewRawDataMsgSpec(subjectName string) (MsgSpec, error) {
	if !SubjectNameRegexp.MatchString(subjectName) {
		return nil, fmt.Errorf("SubjectName format invalid")
	}
	return rawDataMsgSpec(subjectName), nil
}

func (spec rawDataMsgSpec) SubjectName() string {
	return string(spec)
}

func (spec rawDataMsgSpec) NewMsg() interface{} {
	return &rawenc.RawData{}
}

var (
	rawDataType  = reflect.TypeOf((*rawenc.RawData)(nil))
	rawDataValue = &rawenc.RawData{}
)

func (spec rawDataMsgSpec) MsgType() reflect.Type {
	return rawDataType
}

func (spec rawDataMsgSpec) MsgValue() interface{} {
	return rawDataValue
}

func (spec rawDataMsgSpec) String() string {
	return fmt.Sprintf("RawDataMsgSpec(%s)", string(spec))
}

// AssertMsgType makes sure msg's type conform to the spec:
// reflect.TypeOf(msg) == spec.MsgType()
func AssertMsgType(spec MsgSpec, msg interface{}) error {
	if msgType := reflect.TypeOf(msg); msgType != spec.MsgType() {
		return fmt.Errorf("%s got unexpected msg type %s", spec, msgType.String())
	}
	return nil
}
