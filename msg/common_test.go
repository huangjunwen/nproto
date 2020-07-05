package msg

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMsgSpec(t *testing.T) {
	assert := assert.New(t)

	newSpec := func(newMsg func() interface{}) *MsgSpec {
		return &MsgSpec{
			SubjectName: "test",
			NewMsg:      newMsg,
		}
	}

	for i, testCase := range []*struct {
		Spec        *MsgSpec
		Msg         interface{}
		ExpectPanic bool
	}{
		// NewMsg() returns nil.
		{
			Spec:        newSpec(func() interface{} { return nil }),
			Msg:         &emptypb.Empty{},
			ExpectPanic: true,
		},
		// NewMsg() returns non-pointer.
		{
			Spec:        newSpec(func() interface{} { return 0 }),
			Msg:         &emptypb.Empty{},
			ExpectPanic: true,
		},
		// Not the same type.
		{
			Spec:        newSpec(func() interface{} { return wrapperspb.String("") }),
			Msg:         &emptypb.Empty{},
			ExpectPanic: true,
		},
		// Ok.
		{
			Spec:        newSpec(func() interface{} { return wrapperspb.String("") }),
			Msg:         wrapperspb.String("1"),
			ExpectPanic: false,
		},
	} {
		f := assert.NotPanics
		if testCase.ExpectPanic {
			f = assert.Panics
		}
		f(func() {
			defer func() {
				err := recover()
				if err != nil {
					log.Println(err)
					panic(err)
				}
			}()
			testCase.Spec.AssertMsgType(testCase.Msg)
		}, "test case %d", i)
	}
}
