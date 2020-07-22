package rpc

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/huangjunwen/nproto/v2/enc/rawenc"
)

func TestNewRPCSpec(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestNewRPCSpec.\n")
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

	{
		spec := MustRPCSpec(
			"test",
			"test",
			func() interface{} { return wrapperspb.String("") },
			func() interface{} { return wrapperspb.Bool(false) },
		)
		assert.Equal("test", spec.SvcName())
		assert.Equal("test", spec.MethodName())
		assert.True(proto.Equal(wrapperspb.String(""), spec.NewInput().(proto.Message)))
		assert.True(proto.Equal(wrapperspb.Bool(false), spec.NewOutput().(proto.Message)))
		{
			_, ok := spec.InputValue().(*wrapperspb.StringValue)
			assert.True(ok)
		}
		{
			_, ok := spec.OutputValue().(*wrapperspb.BoolValue)
			assert.True(ok)
		}
		assert.NoError(AssertInputType(spec, wrapperspb.String("123")))
		assert.NoError(AssertOutputType(spec, wrapperspb.Bool(true)))
		assert.Error(AssertInputType(spec, wrapperspb.Bool(true)))
		assert.Error(AssertOutputType(spec, wrapperspb.String("123")))
	}

}

func TestNewRawDataRPCSpec(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestNewRawDataRPCSpec.\n")
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

	{
		spec := MustRawDataRPCSpec(
			"test",
			"test",
		)
		assert.Equal("test", spec.SvcName())
		assert.Equal("test", spec.MethodName())
		assert.Equal(&rawenc.RawData{}, spec.NewInput())
		assert.Equal(&rawenc.RawData{}, spec.NewOutput())
		{
			_, ok := spec.InputValue().(*rawenc.RawData)
			assert.True(ok)
		}
		{
			_, ok := spec.OutputValue().(*rawenc.RawData)
			assert.True(ok)
		}
		assert.NoError(AssertInputType(spec, &rawenc.RawData{Format: "json"}))
		assert.NoError(AssertOutputType(spec, &rawenc.RawData{Format: "json"}))
		assert.Error(AssertInputType(spec, 3))
		assert.Error(AssertOutputType(spec, 3))
	}
}
