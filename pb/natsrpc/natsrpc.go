package nppbnatsrpc

import (
	"fmt"

	npmd "github.com/huangjunwen/nproto/v2/md"
	nppbmd "github.com/huangjunwen/nproto/v2/pb/md"
	nprpc "github.com/huangjunwen/nproto/v2/rpc"
)

func AssembleRequest(md npmd.MD, inputFormat string, inputBytes []byte, timeout uint64) *Request {
	return &Request{
		MetaData:    nppbmd.NewMetaData(md),
		InputFormat: inputFormat,
		InputBytes:  inputBytes,
		Timeout:     timeout,
	}
}

func DisassembleRequest(req *Request) (md nppbmd.MetaData, inputFormat string, inputBytes []byte, timeout uint64) {
	return req.MetaData, req.InputFormat, req.InputBytes, req.Timeout
}

func AssembleResponse(typ Response_ResponseType, outputFormat string, outputBytes []byte, errCode nprpc.RPCErrorCode, errMessage string) *Response {
	switch typ {
	case Response_Output:
		return &Response{
			Type:         typ,
			OutputFormat: outputFormat,
			OutputBytes:  outputBytes,
		}

	case Response_RpcErr:
		return &Response{
			Type:       typ,
			ErrCode:    int32(errCode),
			ErrMessage: errMessage,
		}

	case Response_Err:
		return &Response{
			Type:       typ,
			ErrMessage: errMessage,
		}

	default:
		panic(fmt.Errorf("Impossible branch"))
	}
}

func DisassembleResponse(resp *Response) (typ Response_ResponseType, outputFormat string, outputBytes []byte, errCode nprpc.RPCErrorCode, errMessage string) {
	switch resp.Type {
	case Response_Output:
		outputFormat = resp.OutputFormat
		outputBytes = resp.OutputBytes

	case Response_RpcErr:
		errCode = nprpc.RPCErrorCode(resp.ErrCode)
		fallthrough

	case Response_Err:
		errMessage = resp.ErrMessage

	default:
		panic(fmt.Errorf("Impossible branch"))
	}

	typ = resp.Type
	return
}
