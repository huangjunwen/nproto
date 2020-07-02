package pb

import (
	"github.com/huangjunwen/nproto/v2"
	npenc "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
)

func (err *Error) From(in *nproto.Error) {
	err.Reset()
	err.Code = int32(in.Code)
	err.Message = in.Message
}

func (err *Error) To() *nproto.Error {
	return &nproto.Error{
		Code:    nproto.ErrorCode(err.Code),
		Message: err.Message,
	}
}

func (md *MD) From(in npmd.MD) {
	md.Reset()
	in.Keys(func(key string) error {
		md.KeyValues[key] = &ValueList{
			Values: in.Values(key),
		}
		return nil
	})
}

func (md *MD) To() npmd.MetaData {
	ret := npmd.MetaData{}
	for key, valueList := range md.KeyValues {
		ret[key] = valueList.Values
	}
	return ret
}

func (data *RawData) From(in *npenc.RawData) {
	data.EncoderName = in.EncoderName
	data.Bytes = in.Bytes
}

func (data *RawData) To() *npenc.RawData {
	return &npenc.RawData{
		EncoderName: data.EncoderName,
		Bytes:       data.Bytes,
	}
}
