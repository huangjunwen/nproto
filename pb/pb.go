package pb

import (
	"github.com/huangjunwen/nproto/v2"
	npenc "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
)

// From fills Error from nproto.Error.
func (err *Error) From(in *nproto.Error) {
	err.Reset()
	err.Code = int32(in.Code)
	err.Message = in.Message
}

// To converts Error to nproto.Error.
func (err *Error) To() *nproto.Error {
	return &nproto.Error{
		Code:    nproto.ErrorCode(err.Code),
		Message: err.Message,
	}
}

// From fills MD from npmd.MD.
func (md *MD) From(in npmd.MD) {
	md.Reset()
	in.Keys(func(key string) error {
		md.KeyValues[key] = &ValueList{
			Values: in.Values(key),
		}
		return nil
	})
}

// To converts MD to npmd.MetaData.
func (md *MD) To() npmd.MetaData {
	ret := npmd.MetaData{}
	for key, valueList := range md.KeyValues {
		ret[key] = valueList.Values
	}
	return ret
}

func (payload *RawPayload) From(in *npenc.RawPayload) {
	payload.EncoderName = in.EncoderName
	payload.Data = in.Data
}

func (payload *RawPayload) To() *npenc.RawPayload {
	return &npenc.RawPayload{
		EncoderName: payload.EncoderName,
		Data:        payload.Data,
	}
}
