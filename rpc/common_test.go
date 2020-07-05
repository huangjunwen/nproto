package rpc

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRPCSepcInput(t *testing.T) {
	assert := assert.New(t)

	newSpec := func(newInput func() interface{}) *RPCSpec {
		return &RPCSpec{
			SvcName:    "test",
			MethodName: "test",
			NewInput:   newInput,
			NewOutput:  func() interface{} { return &emptypb.Empty{} },
		}
	}

	for i, testCase := range []*struct {
		Spec        *RPCSpec
		Input       interface{}
		ExpectPanic bool
	}{
		// NewInput() returns nil.
		{
			Spec:        newSpec(func() interface{} { return nil }),
			Input:       &emptypb.Empty{},
			ExpectPanic: true,
		},
		// NewInput() returns non-pointer.
		{
			Spec:        newSpec(func() interface{} { return 0 }),
			Input:       &emptypb.Empty{},
			ExpectPanic: true,
		},
		// Not the same type.
		{
			Spec:        newSpec(func() interface{} { return wrapperspb.String("") }),
			Input:       &emptypb.Empty{},
			ExpectPanic: true,
		},
		// Ok.
		{
			Spec:        newSpec(func() interface{} { return wrapperspb.String("") }),
			Input:       wrapperspb.String("1"),
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
			testCase.Spec.AssertInputType(testCase.Input)
		}, "test case %d", i)
	}
}

func TestRPCSepcOutput(t *testing.T) {
	assert := assert.New(t)

	newSpec := func(newOutput func() interface{}) *RPCSpec {
		return &RPCSpec{
			SvcName:    "test",
			MethodName: "test",
			NewInput:   func() interface{} { return &emptypb.Empty{} },
			NewOutput:  newOutput,
		}
	}

	for i, testCase := range []*struct {
		Spec        *RPCSpec
		Output      interface{}
		ExpectPanic bool
	}{
		// NewOutput() returns nil.
		{
			Spec:        newSpec(func() interface{} { return nil }),
			Output:      &emptypb.Empty{},
			ExpectPanic: true,
		},
		// NewOutput() returns non-pointer.
		{
			Spec:        newSpec(func() interface{} { return 0 }),
			Output:      &emptypb.Empty{},
			ExpectPanic: true,
		},
		// Not the same type.
		{
			Spec:        newSpec(func() interface{} { return wrapperspb.String("") }),
			Output:      &emptypb.Empty{},
			ExpectPanic: true,
		},
		// Ok.
		{
			Spec:        newSpec(func() interface{} { return wrapperspb.String("") }),
			Output:      wrapperspb.String("1"),
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
			testCase.Spec.AssertOutputType(testCase.Output)
		}, "test case %d", i)
	}
}
