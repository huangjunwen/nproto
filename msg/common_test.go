package msg

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestMsgSpec(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		Spec                *MsgSpec
		ExpectValidateError bool
	}{
		// Ok.
		{
			Spec: &MsgSpec{
				SubjectName: "test",
				NewMsg:      func() interface{} { return &emptypb.Empty{} },
			},
			ExpectValidateError: false,
		},
		// SvcName empty.
		{
			Spec:                &MsgSpec{},
			ExpectValidateError: true,
		},
		// NewMsg empty.
		{
			Spec: &MsgSpec{
				SubjectName: "test",
			},
			ExpectValidateError: true,
		},
		// NewMsg returns nil.
		{
			Spec: &MsgSpec{
				SubjectName: "test",
				NewMsg:      func() interface{} { return nil },
			},
			ExpectValidateError: true,
		},
		// NewMsg returns non pointer.
		{
			Spec: &MsgSpec{
				SubjectName: "test",
				NewMsg:      func() interface{} { return 0 },
			},
			ExpectValidateError: true,
		},
	} {
		err := testCase.Spec.Validate()
		if testCase.ExpectValidateError {
			assert.Error(err, "test case %d", i)
			assert.False(testCase.Spec.Validated(), "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
			assert.True(testCase.Spec.Validated(), "test case %d", i)
		}

		log.Println(testCase.Spec, err)

	}

	spec := &MsgSpec{
		SubjectName: "test",
		NewMsg:      func() interface{} { return &emptypb.Empty{} },
	}

	assert.Error(spec.AssertMsgType(&emptypb.Empty{}))
	spec.Validate()
	assert.NoError(spec.AssertMsgType(&emptypb.Empty{}))
	assert.Error(spec.AssertMsgType(3))

}
