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
		payload := RawIOMessage(`"abc"`)

		w := &bytes.Buffer{}
		err := Default.EncodePayload(w, payload)
		assert.NoError(err)

		data := w.Bytes()
		assert.Equal([]byte(payload), data)
		fmt.Printf("EncodePayload(%+v): data=%+v, err=%+v\n", payload, data, err)
	}

	// Encode *RawPayload.
	{

		{
			payload := &RawPayload{
				EncoderName: "pb",
				Data:        []byte("\x00\x01\x02"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodePayload(w, payload)
			assert.Error(err)

			fmt.Printf("EncodePayload(%+v): err=%+v\n", payload, err)
		}

		{
			payload := &RawPayload{
				EncoderName: Name,
				Data:        []byte("{1, 2, 3}"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodePayload(w, payload)
			assert.NoError(err)

			datax := w.Bytes()
			assert.Equal(datax, payload.Data)

			fmt.Printf("EncodePayload(%+v): data=%+v, err=%+v\n", payload, datax, err)
		}

	}

	// Encode RawPayload.
	{

		{
			payload := RawPayload{
				EncoderName: "pb",
				Data:        []byte("\x00\x01\x02"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodePayload(w, payload)
			assert.Error(err)

			fmt.Printf("EncodePayload(%+v): err=%+v\n", payload, err)
		}

		{
			payload := RawPayload{
				EncoderName: Name,
				Data:        []byte("{1, 2, 3}"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodePayload(w, payload)
			assert.NoError(err)

			datax := w.Bytes()
			assert.Equal(datax, payload.Data)

			fmt.Printf("EncodePayload(%+v): data=%+v, err=%+v\n", payload, datax, err)
		}
	}

	// Encode proto.Message.
	{
		payload := wrapperspb.String("123")

		w := &bytes.Buffer{}
		err := Default.EncodePayload(w, payload)
		assert.NoError(err)

		data := w.Bytes()
		fmt.Printf("EncodePayload(%+v): data=%+v, err=%+v\n", payload, data, err)
	}

	// Encode other.
	{
		payload := map[string]interface{}{
			"a": []int{1, 2, 3},
			"b": "ccccc",
		}

		w := &bytes.Buffer{}
		err := Default.EncodePayload(w, payload)
		assert.NoError(err)

		data := w.Bytes()
		fmt.Printf("EncodePayload(%+v): data=%+v, err=%+v\n", payload, data, err)
	}

	// Decode JsonIOUnmarshaler.
	data := []byte(`"123"`)
	{
		payload := &RawIOMessage{}

		r := bytes.NewReader(data)
		err := Default.DecodePayload(r, payload)
		assert.NoError(err)

		fmt.Printf("DecodePayload(): payload=%+v, err=%+v\n", payload, err)
	}

	// Decode *RawPayload.
	{
		payload := &RawPayload{}

		r := bytes.NewReader(data)
		err := Default.DecodePayload(r, payload)
		assert.NoError(err)
		assert.Equal(Name, payload.EncoderName)
		assert.Equal(data, payload.Data)

		fmt.Printf("DecodePayload(): payload=%+v, err=%+v\n", payload, err)
	}

	// Decode proto.Message.
	{
		{
			payload := wrapperspb.String("")

			r := bytes.NewReader(data)
			err := Default.DecodePayload(r, payload)
			assert.NoError(err)

			fmt.Printf("DecodePayload(): payload=%+v, err=%+v\n", payload, err)
		}

		{
			payload := wrapperspb.Bool(false)

			r := bytes.NewReader(data)
			err := Default.DecodePayload(r, payload)
			assert.Error(err)

			fmt.Printf("DecodePayload(): payload=%+v, err=%+v\n", payload, err)
		}
	}

	// Decode other.
	{
		s := ""
		payload := &s

		r := bytes.NewReader(data)
		err := Default.DecodePayload(r, payload)
		assert.NoError(err)

		fmt.Printf("DecodePayload(): payload=%+v, err=%+v\n", *payload, err)
	}
}
