package enc

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {

	assert := assert.New(t)

	param := *ptypes.TimestampNow()
	timeout := 10 * time.Nanosecond
	passthru := map[string]string{"a": "z"}

	data := []byte{}
	err := error(nil)

	// Encode.
	{
		req := &RPCRequest{
			Param:    &param,
			Timeout:  &timeout,
			Passthru: passthru,
		}

		data, err = PBClientEncoder{}.EncodeRequest(req)
		assert.NoError(err)
	}

	// Normal decode.
	{
		p := timestamp.Timestamp{}
		req := &RPCRequest{
			Param: &p,
		}
		err = PBServerEncoder{}.DecodeRequest(data, req)
		assert.NoError(err)

		assert.Equal(param.Seconds, p.Seconds)
		assert.Equal(param.Nanos, p.Nanos)
		assert.Equal(timeout, *req.Timeout)
		assert.Equal(passthru, req.Passthru)
	}

	// Panic if Param not set
	{
		req := &RPCRequest{}
		assert.Panics(func() {
			PBServerEncoder{}.DecodeRequest(data, req)
		})
	}

}
