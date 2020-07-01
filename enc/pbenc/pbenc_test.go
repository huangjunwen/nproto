package pbenc

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	. "github.com/huangjunwen/nproto/v2/enc"
)

func TestPbEncoder(t *testing.T) {
	assert := assert.New(t)

	// Encode proto.Message.
	var data []byte
	{
		payload := wrapperspb.String("123")

		w := &bytes.Buffer{}
		err := Default.EncodePayload(w, payload)
		assert.NoError(err)

		data = w.Bytes()
		fmt.Printf("EncodePayload(%+v): data=%+v, err=%+v\n", payload, data, err)
	}

	// Encode *RawPayload.
	{
		{
			payload := &RawPayload{
				EncoderName: "json",
				Data:        []byte("{}"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodePayload(w, payload)
			assert.Error(err)

			fmt.Printf("EncodePayload(%+v): err=%+v\n", payload, err)
		}

		{
			payload := &RawPayload{
				EncoderName: Name,
				Data:        data,
			}

			w := &bytes.Buffer{}
			err := Default.EncodePayload(w, payload)
			assert.NoError(err)

			datax := w.Bytes()
			assert.Equal(data, datax)

			fmt.Printf("EncodePayload(%+v): data=%+v, err=%+v\n", payload, datax, err)
		}
	}

	// Encode RawPayload.
	{
		{
			payload := RawPayload{
				EncoderName: "json",
				Data:        []byte("{}"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodePayload(w, payload)
			assert.Error(err)

			fmt.Printf("EncodePayload(%+v): err=%+v\n", payload, err)
		}

		{
			payload := RawPayload{
				EncoderName: Name,
				Data:        data,
			}

			w := &bytes.Buffer{}
			err := Default.EncodePayload(w, payload)
			assert.NoError(err)

			datax := w.Bytes()
			assert.Equal(data, datax)

			fmt.Printf("EncodePayload(%+v): data=%+v, err=%+v\n", payload, datax, err)
		}
	}

	// Encode non proto.Message.
	{
		payload := 3

		w := &bytes.Buffer{}
		err := Default.EncodePayload(w, payload)
		assert.Error(err)

		fmt.Printf("EncodePayload(%+v): err=%+v\n", payload, err)
	}

	// Decode to proto.Message.
	{
		payload := wrapperspb.String("")

		r := bytes.NewReader(data)
		err := Default.DecodePayload(r, payload)
		assert.NoError(err)

		fmt.Printf("DecodePayload(): payload=%+v, err=%+v\n", payload, err)
	}

	// Decode to *RawPayload.
	{
		payload := &RawPayload{}

		r := bytes.NewReader(data)
		err := Default.DecodePayload(r, payload)
		assert.NoError(err)
		assert.Equal(Name, payload.EncoderName)
		assert.Equal(data, payload.Data)

		fmt.Printf("DecodePayload(): payload=%+v, err=%+v\n", payload, err)
	}

	// Decode non proto.Message.
	{
		var payload string

		r := bytes.NewReader(data)
		err := Default.DecodePayload(r, &payload)
		assert.Error(err)

		fmt.Printf("DecodePayload(): payload=%+v, err=%+v\n", &payload, err)
	}

}
