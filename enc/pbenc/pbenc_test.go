package pbenc

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/huangjunwen/nproto/v2/enc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func EncodedProtoMessage(m proto.Message) string {
	b, err := proto.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func ProtoMessageEqual(left, right interface{}) bool {
	return proto.Equal(left.(proto.Message), right.(proto.Message))
}

func NewString(v string) *string {
	return &v
}

func TestPbEncoder(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		Data              interface{}
		ExpectError       bool
		ExpectEncodedData string
	}{
		// proto.Message
		{
			Data:              wrapperspb.String("123"),
			ExpectError:       false,
			ExpectEncodedData: EncodedProtoMessage(wrapperspb.String("123")),
		},
		// *RawData
		{
			Data: &RawData{
				EncoderName: "non-pb",
				Bytes:       []byte("{}"),
			},
			ExpectError: true,
		},
		// *RawData
		{
			Data: &RawData{
				EncoderName: Default.EncoderName(),
				Bytes:       []byte(EncodedProtoMessage(wrapperspb.Bool(true))),
			},
			ExpectError:       false,
			ExpectEncodedData: EncodedProtoMessage(wrapperspb.Bool(true)),
		},
		// RawData
		{
			Data: RawData{
				EncoderName: "non-pb",
				Bytes:       []byte("{}"),
			},
			ExpectError: true,
		},
		// RawData
		{
			Data: RawData{
				EncoderName: Default.EncoderName(),
				Bytes:       []byte(EncodedProtoMessage(wrapperspb.Bool(true))),
			},
			ExpectError:       false,
			ExpectEncodedData: EncodedProtoMessage(wrapperspb.Bool(true)),
		},
		// other
		{
			Data:        3,
			ExpectError: true,
		},
	} {

		w := &strings.Builder{}
		err := Default.EncodeData(w, testCase.Data)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
			assert.Equal("", w.String(), "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
			assert.Equal(testCase.ExpectEncodedData, w.String(), "test case %d", i)
		}

		fmt.Printf("EncodeData(%+v): result=%+q, err=%v\n", testCase.Data, w.String(), err)
	}

	for i, testCase := range []*struct {
		EncodedData string
		Data        interface{}
		ExpectError bool
		ExpectData  interface{}
		AlterEqual  func(interface{}, interface{}) bool
	}{
		// proto.Message
		{
			EncodedData: EncodedProtoMessage(wrapperspb.String("321")),
			Data:        wrapperspb.String(""),
			ExpectError: false,
			ExpectData:  wrapperspb.String("321"),
			AlterEqual:  ProtoMessageEqual,
		},
		/*
			// proto.Message
			{
				EncodedData: EncodedProtoMessage(wrapperspb.String("x321")),
				Data:        &emptypb.Empty{},
				ExpectError: true,
			},
		*/
		// *RawData
		{
			EncodedData: EncodedProtoMessage(wrapperspb.String("321")),
			Data:        &RawData{},
			ExpectError: false,
			ExpectData: &RawData{
				EncoderName: Default.Name,
				Bytes:       []byte(EncodedProtoMessage(wrapperspb.String("321"))),
			},
		},
		// other
		{
			EncodedData: EncodedProtoMessage(wrapperspb.String("321")),
			Data:        NewString(""),
			ExpectError: true,
		},
	} {

		r := strings.NewReader(testCase.EncodedData)
		err := Default.DecodeData(r, testCase.Data)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
			if testCase.AlterEqual != nil {
				assert.True(testCase.AlterEqual(testCase.ExpectData, testCase.Data), "test case %d", i)
			} else {
				assert.Equal(testCase.ExpectData, testCase.Data, "test case %d", i)
			}
		}

		fmt.Printf("DecodeData(%q): result=%+v, err=%v\n", testCase.EncodedData, testCase.Data, err)
	}

}
