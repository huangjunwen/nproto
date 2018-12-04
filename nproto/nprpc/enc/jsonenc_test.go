package enc

import (
	"errors"
	"testing"
	"time"

	"github.com/huangjunwen/nproto/nproto"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestJSONRequest(t *testing.T) {

	assert := assert.New(t)

	param := ptypes.TimestampNow()
	md := nproto.NewMetaDataPairs("a", "z")
	timeout := 10 * time.Nanosecond

	data := []byte{}
	err := error(nil)

	// Encode.
	{
		req := &RPCRequest{
			Param:    param,
			MetaData: md,
			Timeout:  timeout,
		}

		data, err = JSONClientEncoder{}.EncodeRequest(req)
		assert.NoError(err)
	}

	// Normal decode.
	{
		p := timestamp.Timestamp{}
		req := &RPCRequest{
			Param: &p,
		}
		err = JSONServerEncoder{}.DecodeRequest(data, req)
		assert.NoError(err)

		assert.Equal(param.Seconds, p.Seconds)
		assert.Equal(param.Nanos, p.Nanos)
		assert.Equal(md, req.MetaData)
		assert.Equal(timeout, req.Timeout)
	}

	// Panic if Param not set
	{
		req := &RPCRequest{}
		assert.Panics(func() {
			JSONServerEncoder{}.DecodeRequest(data, req)
		})
	}

}

func TestJSONReply(t *testing.T) {

	assert := assert.New(t)

	// Normal result.
	{
		result := ptypes.TimestampNow()

		data := []byte{}
		err := error(nil)

		// Encode.
		{
			reply := &RPCReply{
				Result: result,
			}
			data, err = JSONServerEncoder{}.EncodeReply(reply)
			assert.NoError(err)
		}

		// Normal decode.
		{
			r := timestamp.Timestamp{}
			reply := &RPCReply{
				Result: &r,
			}
			err = JSONClientEncoder{}.DecodeReply(data, reply)
			assert.NoError(err)

			assert.Equal(r.Seconds, result.Seconds)
			assert.Equal(r.Nanos, result.Nanos)
			assert.Nil(reply.Error)
		}

	}

	// Error result.
	{
		errResult := errors.New("Some error")

		data := []byte{}
		err := error(nil)

		// Encode.
		{
			reply := &RPCReply{
				Error: errResult,
			}
			data, err = JSONServerEncoder{}.EncodeReply(reply)
			assert.NoError(err)
		}

		// Normal decode.
		{
			r := timestamp.Timestamp{}
			reply := &RPCReply{
				Result: &r,
			}
			err = JSONClientEncoder{}.DecodeReply(data, reply)
			assert.NoError(err)

			assert.Equal(reply.Error.Error(), errResult.Error())
			assert.Nil(reply.Result)

		}

	}

}

func BenchmarkJSONEncode(b *testing.B) {

	param := ptypes.TimestampNow()
	md := nproto.NewMetaDataPairs("a", "z")
	timeout := 10 * time.Nanosecond
	req := &RPCRequest{
		Param:    param,
		MetaData: md,
		Timeout:  timeout,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		JSONClientEncoder{}.EncodeRequest(req)
	}

}

func BenchmarkJSONDecode(b *testing.B) {

	param := ptypes.TimestampNow()
	md := nproto.NewMetaDataPairs("a", "z")
	timeout := 10 * time.Nanosecond
	data, _ := JSONClientEncoder{}.EncodeRequest(&RPCRequest{
		Param:    param,
		MetaData: md,
		Timeout:  timeout,
	})

	req := &RPCRequest{
		Param: &timestamp.Timestamp{},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		JSONServerEncoder{}.DecodeRequest(data, req)
	}

}
