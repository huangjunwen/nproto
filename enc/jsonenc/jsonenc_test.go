package jsonenc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	. "github.com/huangjunwen/nproto/v2/enc"
)

type RawIOMessage json.RawMessage

var (
	_ JsonIOMarshaler   = RawIOMessage{}
	_ JsonIOMarshaler   = (*RawIOMessage)(nil)
	_ JsonIOUnmarshaler = (*RawIOMessage)(nil)
)

func NewRawIOMessage(s string) *RawIOMessage {
	ret := RawIOMessage(s)
	return &ret
}

func (m RawIOMessage) MarshalJSONToWriter(w io.Writer) error {
	_, err := w.Write([]byte(m))
	return err
}

func (m *RawIOMessage) UnmarshalJSONFromReader(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	*m = RawIOMessage(b)
	return nil
}

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
		ExpectEncodedData string
	}{
		// JsonIOMarshaler
		{
			Data:              NewRawIOMessage(`"abc"`),
			ExpectError:       false,
			ExpectEncodedData: `"abc"`,
		},
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
			ExpectEncodedData: "{1,2,3}",
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
			ExpectEncodedData: "{1,2,3}",
		},
		// proto.Message
		{
			Data:              wrapperspb.String("123"),
			ExpectError:       false,
			ExpectEncodedData: `"123"`,
		},
		// other
		{
			Data:              map[string]interface{}{"a": []int{1, 2, 3}},
			ExpectError:       false,
			ExpectEncodedData: `{"a":[1,2,3]}`,
		},
		// other
		{
			Data:        map[string]interface{}{"a": func() {}},
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
		// JsonIOUnmarshaler
		{
			EncodedData: `"123"`,
			Data:        NewRawIOMessage(""),
			ExpectError: false,
			ExpectData:  NewRawIOMessage(`"123"`),
		},
		// *RawData
		{
			EncodedData: `"123"`,
			Data:        &RawData{},
			ExpectError: false,
			ExpectData: &RawData{
				EncoderName: Default.EncoderName(),
				Bytes:       []byte(`"123"`),
			},
		},
		// proto.Message
		{
			EncodedData: `"123"`,
			Data:        wrapperspb.String(""),
			ExpectError: false,
			ExpectData:  wrapperspb.String("123"),
			AlterEqual:  ProtoMessageEqual,
		},
		// proto.Message
		{
			EncodedData: `"123"`,
			Data:        wrapperspb.Bool(false),
			ExpectError: true,
		},
		// other
		{
			EncodedData: `"123"`,
			Data:        NewString(""),
			ExpectError: false,
			ExpectData:  NewString("123"),
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
