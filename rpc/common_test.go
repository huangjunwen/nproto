package rpc

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/huangjunwen/nproto/v2/enc"
)

func TestNewRPCSpec(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		SvcName     string
		MethodName  string
		NewInput    func() interface{}
		NewOutput   func() interface{}
		ExpectError bool
	}{
		// Ok.
		{
			SvcName:     "a-b-c",
			MethodName:  "a-b-c",
			NewInput:    func() interface{} { return &emptypb.Empty{} },
			NewOutput:   func() interface{} { return &emptypb.Empty{} },
			ExpectError: false,
		},
		// Empty SvcName.
		{
			SvcName:     "",
			MethodName:  "test",
			NewInput:    func() interface{} { return &emptypb.Empty{} },
			NewOutput:   func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// Invalid SvcName.
		{
			SvcName:     "a.b",
			MethodName:  "test",
			NewInput:    func() interface{} { return &emptypb.Empty{} },
			NewOutput:   func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// Empty MethodName.
		{
			SvcName:     "test",
			MethodName:  "",
			NewInput:    func() interface{} { return &emptypb.Empty{} },
			NewOutput:   func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// Invalid MethodName.
		{
			SvcName:     "test",
			MethodName:  "a.b",
			NewInput:    func() interface{} { return &emptypb.Empty{} },
			NewOutput:   func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// Nil NewInput.
		{
			SvcName:     "test",
			MethodName:  "test",
			NewInput:    nil,
			NewOutput:   func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// NewInput() returns nil.
		{
			SvcName:     "test",
			MethodName:  "test",
			NewInput:    func() interface{} { return nil },
			NewOutput:   func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// NewInput() returns non pointer.
		{
			SvcName:     "test",
			MethodName:  "test",
			NewInput:    func() interface{} { return 3 },
			NewOutput:   func() interface{} { return &emptypb.Empty{} },
			ExpectError: true,
		},
		// Nil NewOutput.
		{
			SvcName:     "test",
			MethodName:  "test",
			NewInput:    func() interface{} { return &emptypb.Empty{} },
			NewOutput:   nil,
			ExpectError: true,
		},
		// NewOutput() returns nil.
		{
			SvcName:     "test",
			MethodName:  "test",
			NewInput:    func() interface{} { return &emptypb.Empty{} },
			NewOutput:   func() interface{} { return nil },
			ExpectError: true,
		},
		// NewOutput() returns non pointer.
		{
			SvcName:     "test",
			MethodName:  "test",
			NewInput:    func() interface{} { return &emptypb.Empty{} },
			NewOutput:   func() interface{} { return 3 },
			ExpectError: true,
		},
	} {

		spec, err := NewRPCSpec(
			testCase.SvcName,
			testCase.MethodName,
			testCase.NewInput,
			testCase.NewOutput,
		)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
		}

		log.Println(spec, err)
	}

}

func TestNewRawDataRPCSpec(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		SvcName     string
		MethodName  string
		ExpectError bool
	}{
		// Ok.
		{
			SvcName:     "a-b-c",
			MethodName:  "a-b-c",
			ExpectError: false,
		},
		// Empty SvcName.
		{
			SvcName:     "",
			MethodName:  "test",
			ExpectError: true,
		},
		// Invalid SvcName.
		{
			SvcName:     "a.b",
			MethodName:  "test",
			ExpectError: true,
		},
		// Empty MethodName.
		{
			SvcName:     "test",
			MethodName:  "",
			ExpectError: true,
		},
		// Invalid MethodName.
		{
			SvcName:     "test",
			MethodName:  "a.b",
			ExpectError: true,
		},
	} {

		spec, err := NewRawDataRPCSpec(
			testCase.SvcName,
			testCase.MethodName,
		)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
		}

		log.Println(spec, err)
	}
}

func TestAssertInputType(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		Spec        RPCSpec
		Input       interface{}
		ExpectError bool
	}{
		// Ok.
		{
			Spec: MustRawDataRPCSpec(
				"test",
				"test",
			),
			Input: &enc.RawData{
				EncoderName: "json",
				Bytes:       []byte("{}"),
			},
			ExpectError: false,
		},
		// Failed.
		{
			Spec: MustRPCSpec(
				"test",
				"test",
				func() interface{} { return wrapperspb.String("") },
				func() interface{} { return wrapperspb.String("") },
			),
			Input:       "123",
			ExpectError: true,
		},
	} {

		err := AssertInputType(testCase.Spec, testCase.Input)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
		}

		log.Println(testCase.Spec, testCase.Input, err)
	}
}

func TestAssertOutputType(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		Spec        RPCSpec
		Output      interface{}
		ExpectError bool
	}{
		// Ok.
		{
			Spec: MustRawDataRPCSpec(
				"test",
				"test",
			),
			Output: &enc.RawData{
				EncoderName: "json",
				Bytes:       []byte("{}"),
			},
			ExpectError: false,
		},
		// Failed.
		{
			Spec: MustRPCSpec(
				"test",
				"test",
				func() interface{} { return wrapperspb.String("") },
				func() interface{} { return wrapperspb.String("") },
			),
			Output:      "123",
			ExpectError: true,
		},
	} {

		err := AssertOutputType(testCase.Spec, testCase.Output)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
		}

		log.Println(testCase.Spec, testCase.Output, err)
	}
}
