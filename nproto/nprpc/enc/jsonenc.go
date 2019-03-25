package enc

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang/protobuf/jsonpb"

	"github.com/huangjunwen/nproto/nproto"
)

// JSONServerEncoder is RPCServerEncoder using json encoding.
type JSONServerEncoder struct{}

// JSONClientEncoder is RPCClientEncoder using json encoding.
type JSONClientEncoder struct{}

// JSONRequest is request of a RPC call encoded by json.
type JSONRequest struct {
	Param    json.RawMessage `json:"param"`
	MetaData nproto.MetaData `json:"metaData,omitempty"`
	Timeout  int64           `json:"timeout,omitempty"`
}

// JSONReply is reply of a RPC call encoded by json.
type JSONReply struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

var (
	_ RPCServerEncoder = JSONServerEncoder{}
	_ RPCClientEncoder = JSONClientEncoder{}
)

var (
	jsonUnmarshaler = jsonpb.Unmarshaler{
		AllowUnknownFields: true,
	}
	jsonMarshaler = jsonpb.Marshaler{
		EmitDefaults: true,
	}
)

// DecodeRequest implements RPCServerEncoder interface.
func (e JSONServerEncoder) DecodeRequest(data []byte, req *RPCRequest) error {
	// Decode data.
	r := &JSONRequest{}
	if err := json.Unmarshal(data, r); err != nil {
		return err
	}

	// Decode param.
	reader := bytes.NewReader(r.Param)
	if err := jsonUnmarshaler.Unmarshal(reader, req.Param); err != nil {
		return err
	}

	// Meta data.
	if len(r.MetaData) != 0 {
		req.MD = r.MetaData
	}

	// Timeout.
	if r.Timeout > 0 {
		req.Timeout = time.Duration(r.Timeout)
	}

	return nil

}

// EncodeReply implements RPCServerEncoder interface.
func (e JSONServerEncoder) EncodeReply(reply *RPCReply) ([]byte, error) {

	r := &JSONReply{}
	if reply.Error != nil {
		// Set error.
		r.Error = reply.Error.Error()
	} else {
		// Set result.
		buf := &bytes.Buffer{}
		if err := jsonMarshaler.Marshal(buf, reply.Result); err != nil {
			return nil, err
		}
		r.Result = json.RawMessage(buf.Bytes())
	}

	// Encode reply.
	return json.Marshal(r)
}

// EncodeRequest implements RPCClientEncoder interface.
func (e JSONClientEncoder) EncodeRequest(req *RPCRequest) ([]byte, error) {
	r := &JSONRequest{}

	// Encode param.
	buf := &bytes.Buffer{}
	if err := jsonMarshaler.Marshal(buf, req.Param); err != nil {
		return nil, err
	}
	r.Param = []byte(buf.Bytes())

	// Meta data.
	if req.MD != nil {
		r.MetaData = nproto.NewMetaDataFromMD(req.MD)
	}

	// Timeout.
	if req.Timeout > 0 {
		r.Timeout = int64(req.Timeout)
	}

	// Encode request.
	return json.Marshal(r)
}

// DecodeReply implements RPCClientEncoder interface.
func (e JSONClientEncoder) DecodeReply(data []byte, reply *RPCReply) error {
	// Decode data.
	r := &JSONReply{}
	if err := json.Unmarshal(data, r); err != nil {
		return err
	}

	// If there is an error.
	if r.Error != "" {
		reply.Result = nil
		reply.Error = errors.New(r.Error)
		return nil
	}

	// Decode result.
	reader := bytes.NewReader([]byte(r.Result))
	return jsonUnmarshaler.Unmarshal(reader, reply.Result)

}
