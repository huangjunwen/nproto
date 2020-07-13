package jsonenc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	. "github.com/huangjunwen/nproto/v2/enc"
)

func NewString(v string) *string {
	return &v
}

func ProtoMessageEqual(left, right interface{}) bool {
	return proto.Equal(left.(proto.Message), right.(proto.Message))
}

func TestJsonEncoder(t *testing.T) {
	assert := assert.New(t)

	for i, testCase := range []*struct {
		Data              interface{}
		ExpectError       bool
		ExpectEncodedData []byte
	}{
		// *RawData
		{
			Data: &RawData{
				EncoderName: "not-json",
				Bytes:       []byte("\x00\x01\x02"),
			},
			ExpectError: true,
		},
		// *RawData
		{
			Data: &RawData{
				EncoderName: Default.EncoderName(),
				Bytes:       []byte("{1,2,3}"),
			},
			ExpectError:       false,
			ExpectEncodedData: []byte("{1,2,3}"),
		},
		// RawData
		{
			Data: RawData{
				EncoderName: "not-json",
				Bytes:       []byte("\x00\x01\x02"),
			},
			ExpectError: true,
		},
		// RawData
		{
			Data: RawData{
				EncoderName: Default.EncoderName(),
				Bytes:       []byte("{1,2,3}"),
			},
			ExpectError:       false,
			ExpectEncodedData: []byte("{1,2,3}"),
		},
		// proto.Message
		{
			Data:              wrapperspb.String("123"),
			ExpectError:       false,
			ExpectEncodedData: []byte(`"123"`),
		},
		// other
		{
			Data:              map[string]interface{}{"a": []int{1, 2, 3}},
			ExpectError:       false,
			ExpectEncodedData: []byte(`{"a":[1,2,3]}`),
		},
		// other
		{
			Data:        map[string]interface{}{"a": func() {}},
			ExpectError: true,
		},
	} {
		w := []byte{}
		err := Default.EncodeData(testCase.Data, &w)
		if testCase.ExpectError {
			assert.Error(err, "test case %d", i)
			assert.Len(w, 0, "test case %d", i)
		} else {
			assert.NoError(err, "test case %d", i)
			assert.Equal(testCase.ExpectEncodedData, w, "test case %d", i)
		}

		fmt.Printf("EncodeData(%+v): w=%+q, err=%v\n", testCase.Data, w, err)
	}

	for i, testCase := range []*struct {
		EncodedData []byte
		Data        interface{}
		ExpectError bool
		ExpectData  interface{}
		AlterEqual  func(interface{}, interface{}) bool
	}{
		// *RawData
		{
			EncodedData: []byte(`"123"`),
			Data:        &RawData{},
			ExpectError: false,
			ExpectData: &RawData{
				EncoderName: Default.EncoderName(),
				Bytes:       []byte(`"123"`),
			},
		},
		// proto.Message
		{
			EncodedData: []byte(`"123"`),
			Data:        wrapperspb.String(""),
			ExpectError: false,
			ExpectData:  wrapperspb.String("123"),
			AlterEqual:  ProtoMessageEqual,
		},
		// proto.Message
		{
			EncodedData: []byte(`"123"`),
			Data:        wrapperspb.Bool(false),
			ExpectError: true,
		},
		// other
		{
			EncodedData: []byte(`"123"`),
			Data:        NewString(""),
			ExpectError: false,
			ExpectData:  NewString("123"),
		},
	} {

		r := testCase.EncodedData
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

		fmt.Printf("DecodeData(%q): data=%+v, err=%v\n", testCase.EncodedData, testCase.Data, err)
	}

}
