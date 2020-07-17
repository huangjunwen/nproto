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

	for i, testCase := range []*struct {
		SubjectName string
		NewMsg      func() interface{}
		ExpectError bool
	}{
		// Ok.
		{
			SubjectName: "a-b.c-d.e",
			NewMsg:      func() interface{} { return &emptypb.Empty{} },
			ExpectError: false,
		},
		// Empty subject.
		{
			SubjectName: "",
			NewMsg:      func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// Invalid subject.
		{
			SubjectName: "a.b.",
			NewMsg:      func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// Invalid subject.
		{
			SubjectName: ".a.b",
			NewMsg:      func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// Nil NewMsg.
		{
			SubjectName: "a.b",
			NewMsg:      nil,
			ExpectError: true,
		},
		// NewMsg() returns nil.
		{
			SubjectName: "a.b",
			NewMsg:      func() interface{} { return nil },
			ExpectError: true,
		},
		// NewMsg() returns non pointer.
		{
			SubjectName: "a.b",
			NewMsg:      func() interface{} { return 3 },
			ExpectError: true,
		},
	} {
		spec, err := NewMsgSpec(
			testCase.SubjectName,
			testCase.NewMsg,
		)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
		}

		log.Println(spec, err)

	}

}

func TestAssertMsgType(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		Spec        MsgSpec
		Msg         interface{}
		ExpectError bool
	}{
		// Ok.
		{
			Spec: MustMsgSpec(
				"test",
				func() interface{} { return wrapperspb.String("") },
			),
			Msg:         wrapperspb.String("123"),
			ExpectError: false,
		},
		// Failed.
		{
			Spec: MustMsgSpec(
				"test",
				func() interface{} { return wrapperspb.String("") },
			),
			Msg:         "123",
			ExpectError: true,
		},
	} {
		err := AssertMsgType(testCase.Spec, testCase.Msg)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
		}

		log.Println(testCase.Spec, testCase.Msg, err)

	}

}
