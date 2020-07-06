package rpc

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestRPCSepc(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		Spec                *RPCSpec
		ExpectValidateError bool
	}{
		// Ok.
		{
			Spec: &RPCSpec{
				SvcName:    "test",
				MethodName: "test",
				NewInput:   func() interface{} { return &emptypb.Empty{} },
				NewOutput:  func() interface{} { return &emptypb.Empty{} },
			},
			ExpectValidateError: false,
		},
		// RPCSpec use RawData as input/output.
		{
			Spec:                NewRawDataRPCSpec("test", "test),"),
			ExpectValidateError: false,
		},
		// SvcName empty.
		{
			Spec:                &RPCSpec{},
			ExpectValidateError: true,
		},
		// MethodName empty.
		{
			Spec: &RPCSpec{
				SvcName: "test",
			},
			ExpectValidateError: true,
		},
		// NewInput empty.
		{
			Spec: &RPCSpec{
				SvcName:    "test",
				MethodName: "test",
			},
			ExpectValidateError: true,
		},
		// NewOutput empty.
		{
			Spec: &RPCSpec{
				SvcName:    "test",
				MethodName: "test",
				NewInput:   func() interface{} { return &emptypb.Empty{} },
			},
			ExpectValidateError: true,
		},
		// NewOutput returns nil.
		{
			Spec: &RPCSpec{
				SvcName:    "test",
				MethodName: "test",
				NewInput:   func() interface{} { return &emptypb.Empty{} },
				NewOutput:  func() interface{} { return nil },
			},
			ExpectValidateError: true,
		},
		// NewOutput returns non pointer.
		{
			Spec: &RPCSpec{
				SvcName:    "test",
				MethodName: "test",
				NewInput:   func() interface{} { return &emptypb.Empty{} },
				NewOutput:  func() interface{} { return 0 },
			},
			ExpectValidateError: true,
		},
		// NewInput returns nil.
		{
			Spec: &RPCSpec{
				SvcName:    "test",
				MethodName: "test",
				NewInput:   func() interface{} { return nil },
				NewOutput:  func() interface{} { return &emptypb.Empty{} },
			},
			ExpectValidateError: true,
		},
		// NewInput returns non pointer.
		{
			Spec: &RPCSpec{
				SvcName:    "test",
				MethodName: "test",
				NewInput:   func() interface{} { return 0 },
				NewOutput:  func() interface{} { return &emptypb.Empty{} },
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

	spec := &RPCSpec{
		SvcName:    "test",
		MethodName: "test",
		NewInput:   func() interface{} { return &emptypb.Empty{} },
		NewOutput:  func() interface{} { return &emptypb.Empty{} },
	}

	assert.Error(spec.AssertInputType(&emptypb.Empty{}))
	spec.Validate()
	assert.NoError(spec.AssertInputType(&emptypb.Empty{}))
	assert.Error(spec.AssertInputType(3))

}
