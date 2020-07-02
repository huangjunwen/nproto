package jsonenc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	. "github.com/huangjunwen/nproto/v2/enc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type RawIOMessage json.RawMessage

var (
	_ JsonIOMarshaler   = RawIOMessage{}
	_ JsonIOUnmarshaler = (*RawIOMessage)(nil)
)

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

func TestJsonEncoder(t *testing.T) {
	assert := assert.New(t)

	// Encode JsonIOMarshaler.
	{
		data := RawIOMessage(`"abc"`)

		w := &bytes.Buffer{}
		err := Default.EncodeData(w, data)
		assert.NoError(err)
		assert.Equal([]byte(data), w.Bytes())

		fmt.Printf("EncodeData(%+v): bytes=%+v, err=%v\n", data, w.Bytes(), err)
	}

	// Encode *RawData.
	{

		{
			data := &RawData{
				EncoderName: "pb",
				Bytes:       []byte("\x00\x01\x02"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodeData(w, data)
			assert.Error(err)

			fmt.Printf("EncodeData(%+v): err=%v\n", data, err)
		}

		{
			data := &RawData{
				EncoderName: Name,
				Bytes:       []byte("{1, 2, 3}"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodeData(w, data)
			assert.NoError(err)
			assert.Equal(w.Bytes(), data.Bytes)

			fmt.Printf("EncodeData(%+v): bytes=%+v, err=%v\n", data, w.Bytes(), err)
		}

	}

	// Encode RawData.
	{

		{
			data := RawData{
				EncoderName: "pb",
				Bytes:       []byte("\x00\x01\x02"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodeData(w, data)
			assert.Error(err)

			fmt.Printf("EncodeData(%+v): err=%v\n", data, err)
		}

		{
			data := RawData{
				EncoderName: Name,
				Bytes:       []byte("{1, 2, 3}"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodeData(w, data)
			assert.NoError(err)
			assert.Equal(w.Bytes(), data.Bytes)

			fmt.Printf("EncodeData(%+v): bytes=%+v, err=%v\n", data, w.Bytes(), err)
		}

	}

	// Encode proto.Message.
	{
		data := wrapperspb.String("123")

		w := &bytes.Buffer{}
		err := Default.EncodeData(w, data)
		assert.NoError(err)

		fmt.Printf("EncodeData(%+v): bytes=%+v, err=%v\n", data, w.Bytes(), err)
	}

	// Encode other.
	{
		data := map[string]interface{}{
			"a": []int{1, 2, 3},
			"b": "ccccc",
		}

		w := &bytes.Buffer{}
		err := Default.EncodeData(w, data)
		assert.NoError(err)

		fmt.Printf("EncodeData(%+v): bytes=%+v, err=%v\n", data, w.Bytes(), err)
	}

	// Decode JsonIOUnmarshaler.
	b := []byte(`"123"`)
	{
		data := &RawIOMessage{}

		r := bytes.NewReader(b)
		err := Default.DecodeData(r, data)
		assert.NoError(err)

		fmt.Printf("DecodeData(): data=%+v, err=%v\n", data, err)
	}

	// Decode *RawData.
	{
		data := &RawData{}

		r := bytes.NewReader(b)
		err := Default.DecodeData(r, data)
		assert.NoError(err)
		assert.Equal(Name, data.EncoderName)
		assert.Equal(b, data.Bytes)

		fmt.Printf("DecodeData(): data=%+v, err=%v\n", data, err)
	}

	// Decode proto.Message.
	{
		{
			data := wrapperspb.String("")

			r := bytes.NewReader(b)
			err := Default.DecodeData(r, data)
			assert.NoError(err)

			fmt.Printf("DecodeData(): data=%+v, err=%v\n", data, err)
		}

		{
			data := wrapperspb.Bool(false)

			r := bytes.NewReader(b)
			err := Default.DecodeData(r, data)
			assert.Error(err)

			fmt.Printf("DecodeData(): data=%+v, err=%v\n", data, err)
		}
	}

	// Decode other.
	{
		s := ""
		data := &s

		r := bytes.NewReader(b)
		err := Default.DecodeData(r, data)
		assert.NoError(err)

		fmt.Printf("DecodeData(): data=%+v, err=%v\n", *data, err)
	}
}
