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
	var b []byte
	{
		data := wrapperspb.String("123")

		w := &bytes.Buffer{}
		err := Default.EncodeData(w, data)
		assert.NoError(err)

		b = w.Bytes()
		fmt.Printf("EncodeData(%+v): bytes=%+v, err=%+v\n", data, b, err)
	}

	// Encode *RawData.
	{
		{
			data := &RawData{
				EncoderName: "json",
				Bytes:       []byte("{}"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodeData(w, data)
			assert.Error(err)

			fmt.Printf("EncodeData(%+v): err=%+v\n", data, err)
		}

		{
			data := &RawData{
				EncoderName: Name,
				Bytes:       b,
			}

			w := &bytes.Buffer{}
			err := Default.EncodeData(w, data)
			assert.NoError(err)
			assert.Equal(b, w.Bytes())

			fmt.Printf("EncodeData(%+v): bytes=%+v, err=%+v\n", data, w.Bytes(), err)
		}
	}

	// Encode RawData.
	{
		{
			data := RawData{
				EncoderName: "json",
				Bytes:       []byte("{}"),
			}

			w := &bytes.Buffer{}
			err := Default.EncodeData(w, data)
			assert.Error(err)

			fmt.Printf("EncodeData(%+v): err=%+v\n", data, err)
		}

		{
			data := RawData{
				EncoderName: Name,
				Bytes:       b,
			}

			w := &bytes.Buffer{}
			err := Default.EncodeData(w, data)
			assert.NoError(err)
			assert.Equal(b, w.Bytes())

			fmt.Printf("EncodeData(%+v): bytes=%+v, err=%+v\n", data, w.Bytes(), err)
		}
	}

	// Encode non proto.Message.
	{
		data := 3

		w := &bytes.Buffer{}
		err := Default.EncodeData(w, data)
		assert.Error(err)

		fmt.Printf("EncodeData(%+v): err=%+v\n", data, err)
	}

	// Decode to proto.Message.
	{
		data := wrapperspb.String("")

		r := bytes.NewReader(b)
		err := Default.DecodeData(r, data)
		assert.NoError(err)

		fmt.Printf("DecodeData(): data=%+v, err=%+v\n", data, err)
	}

	// Decode to *RawData.
	{
		data := &RawData{}

		r := bytes.NewReader(b)
		err := Default.DecodeData(r, data)
		assert.NoError(err)
		assert.Equal(Name, data.EncoderName)
		assert.Equal(b, data.Bytes)

		fmt.Printf("DecodeData(): data=%+v, err=%+v\n", data, err)
	}

	// Decode non proto.Message.
	{
		var data string

		r := bytes.NewReader(b)
		err := Default.DecodeData(r, &data)
		assert.Error(err)

		fmt.Printf("DecodeData(): data=%+v, err=%+v\n", &data, err)
	}

}
