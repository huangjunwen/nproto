package msg

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/huangjunwen/nproto/v2/enc/rawenc"
)

func TestNewMsgSpec(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestNewMsgSpec.\n")
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

	{
		spec := MustMsgSpec(
			"test",
			func() interface{} { return wrapperspb.String("") },
		)
		assert.Equal("test", spec.SubjectName())
		assert.True(proto.Equal(wrapperspb.String(""), spec.NewMsg().(proto.Message)))
		{
			_, ok := spec.MsgValue().(*wrapperspb.StringValue)
			assert.True(ok)
		}
		assert.NoError(AssertMsgType(spec, wrapperspb.String("123")))
		assert.Error(AssertMsgType(spec, wrapperspb.Bool(true)))
	}
}

func TestNewRawDataMsgSpec(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestNewRawDataMsgSpec.\n")
	assert := assert.New(t)

	for i, testCase := range []*struct {
		SubjectName string
		ExpectError bool
	}{
		// Ok.
		{
			SubjectName: "a-b.c-d.e",
			ExpectError: false,
		},
		// Empty subject.
		{
			SubjectName: "",
			ExpectError: true,
		},
		// Invalid subject.
		{
			SubjectName: "a.b.",
			ExpectError: true,
		},
		// Invalid subject.
		{
			SubjectName: ".a.b",
			ExpectError: true,
		},
	} {
		spec, err := NewRawDataMsgSpec(
			testCase.SubjectName,
		)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
		}

		log.Println(spec, err)

	}

	{
		spec := MustRawDataMsgSpec(
			"test",
		)
		assert.Equal("test", spec.SubjectName())
		assert.Equal(&rawenc.RawData{}, spec.NewMsg())
		{
			_, ok := spec.MsgValue().(*rawenc.RawData)
			assert.True(ok)
		}
		assert.NoError(AssertMsgType(spec, &rawenc.RawData{Format: "json"}))
		assert.Error(AssertMsgType(spec, 3))
	}
}
