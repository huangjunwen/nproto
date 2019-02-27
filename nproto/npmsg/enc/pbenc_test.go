package enc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/huangjunwen/nproto/nproto"
)

func TestPBEncodeDecode(t *testing.T) {

	assert := assert.New(t)
	msgData := []byte(`{"hello":"world"}`)
	md := nproto.NewMetaDataPairs("a", "z")

	data := []byte{}
	err := error(nil)

	// Encode.
	{
		p := &MsgPayload{
			MsgData: msgData,
			MD:      md,
		}

		data, err = PBMsgPayloadEncoder{}.EncodePayload(p)
		assert.NoError(err)
	}

	// Decode.
	{
		q := &MsgPayload{}
		err = PBMsgPayloadDecoder{}.DecodePayload(data, q)
		assert.NoError(err)

		md2 := nproto.NewMetaDataFromMD(q.MD)

		assert.Equal(msgData, q.MsgData)
		assert.Equal(md, md2)
	}

}
