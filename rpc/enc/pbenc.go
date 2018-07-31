package enc

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
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
	r := &PBRequest{}
	if err := proto.Unmarshal(data, r); err != nil {
		return err
	}

	// Decode param.
	if err := proto.Unmarshal(r.Param, req.Param); err != nil {
		return err
	}

	// Optional passthru.
	req.Passthru = r.Passthru

	// Optional timeout.
	if r.Timeout != nil {
		dur, err := ptypes.Duration(r.Timeout)
		if err != nil {
			return err
		}
		req.Timeout = &dur
	}

	return nil

}

// EncodeReply implements RPCServerEncoder interface.
func (e PBServerEncoder) EncodeReply(reply *RPCReply) ([]byte, error) {

	r := &PBReply{}
	if reply.Error != nil {
		// Set error.
		r.Reply = &PBReply_Error{
			Error: reply.Error.Error(),
		}
	} else {
		// Encode result.
		result, err := proto.Marshal(reply.Result)
		if err != nil {
			return nil, err
		}
		r.Reply = &PBReply_Result{
			Result: result,
		}
	}

	// Encode reply.
	return proto.Marshal(r)
}

// EncodeRequest implements RPCClientEncoder interface.
func (e PBClientEncoder) EncodeRequest(req *RPCRequest) ([]byte, error) {

	var err error
	r := &PBRequest{}
	// Encode param.
	r.Param, err = proto.Marshal(req.Param)
	if err != nil {
		return nil, err
	}

	// Optional passthru.
	r.Passthru = req.Passthru

	// Optional timeout.
	if req.Timeout != nil {
		r.Timeout = ptypes.DurationProto(*req.Timeout)
	}

	// Encode request.
	return proto.Marshal(r)

}

// DecodeReply implements RPCClientEncoder interface.
func (e PBClientEncoder) DecodeReply(data []byte, reply *RPCReply) error {

	// Decode reply.
	r := &PBReply{}
	if err := proto.Unmarshal(data, r); err != nil {
		return err
	}

	switch x := r.Reply.(type) {
	case *PBReply_Error:
		// Error case.
		reply.Error = errors.New(x.Error)
		reply.Result = nil
		return nil
	case *PBReply_Result:
		// Normal case. Decode result.
		return proto.Unmarshal(x.Result, reply.Result)
	default:
		panic("Shouldn't be here")
	}

}
