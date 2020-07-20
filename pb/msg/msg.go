package nppbmsg

import (
	npmd "github.com/huangjunwen/nproto/v2/md"
	nppbmd "github.com/huangjunwen/nproto/v2/pb/md"
)

func AssembleMessageWithMD(md npmd.MD, msgFormat string, msgBytes []byte) *MessageWithMD {
	return &MessageWithMD{
		MetaData:  nppbmd.NewMetaData(md),
		MsgFormat: msgFormat,
		MsgBytes:  msgBytes,
	}
}

func DisassembleMessageWithMD(src *MessageWithMD) (md nppbmd.MetaData, msgFormat string, msgBytes []byte) {
	return src.MetaData, src.MsgFormat, src.MsgBytes
}
