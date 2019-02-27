package enc

import (
	"errors"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/huangjunwen/nproto/nproto"
	"github.com/huangjunwen/nproto/nproto/pb"
)

// PBServerEncoder is RPCServerEncoder using protobuf encoding.
type PBServerEncoder struct{}

// PBClientEncoder is RPCClientEncoder using protobuf encoding.
type PBClientEncoder struct{}

var (
	_ RPCServerEncoder = PBServerEncoder{}
	_ RPCClientEncoder = PBClientEncoder{}
)

// DecodeRequest implements RPCServerEncoder interface.
func (e PBServerEncoder) DecodeRequest(data []byte, req *RPCRequest) error {
	// Decode request.
	r := &pb.RPCRequest{}
	if err := proto.Unmarshal(data, r); err != nil {
		return err
	}

	// Decode param.
	if err := proto.Unmarshal(r.Param, req.Param); err != nil {
		return err
	}

	// Meta data.
	if len(r.MetaData) != 0 {
		md := nproto.MetaData{}
		for _, kv := range r.MetaData {
			md[kv.Key] = kv.Values
		}
		req.MD = md
	}

	// Timeout.
	req.Timeout = 0
	if r.Timeout > 0 {
		req.Timeout = time.Duration(r.Timeout)
	}

	return nil

}

// EncodeReply implements RPCServerEncoder interface.
func (e PBServerEncoder) EncodeReply(reply *RPCReply) ([]byte, error) {
	r := &pb.RPCReply{}
	if reply.Error != nil {
		// Set error.
		r.Error = reply.Error.Error()
	} else {
		// Set result.
		result, err := proto.Marshal(reply.Result)
		if err != nil {
			return nil, err
		}
		r.Result = result
	}

	// Encode reply.
	return proto.Marshal(r)
}

// EncodeRequest implements RPCClientEncoder interface.
func (e PBClientEncoder) EncodeRequest(req *RPCRequest) ([]byte, error) {
	var err error
	r := &pb.RPCRequest{}
	// Encode param.
	r.Param, err = proto.Marshal(req.Param)
	if err != nil {
		return nil, err
	}

	// Meta data.
	if req.MD != nil {
		req.MD.Keys(func(key string) error {
			r.MetaData = append(r.MetaData, &pb.MetaDataKV{
				Key:    key,
				Values: req.MD.Values(key),
			})
			return nil
		})
	}

	// Timeout.
	if req.Timeout > 0 {
		r.Timeout = int64(req.Timeout)
	}

	// Encode request.
	return proto.Marshal(r)

}

// DecodeReply implements RPCClientEncoder interface.
func (e PBClientEncoder) DecodeReply(data []byte, reply *RPCReply) error {
	// Decode reply.
	r := &pb.RPCReply{}
	if err := proto.Unmarshal(data, r); err != nil {
		return err
	}

	// If there is an error.
	if r.Error != "" {
		reply.Result = nil
		reply.Error = errors.New(r.Error)
		return nil
	}

	// Decode result.
	return proto.Unmarshal(r.Result, reply.Result)

}
