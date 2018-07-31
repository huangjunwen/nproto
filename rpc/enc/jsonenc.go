package enc

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang/protobuf/jsonpb"
)

// JSONServerEncoder is RPCServerEncoder using json encoding.
type JSONServerEncoder struct{}

// JSONClientEncoder is RPCClientEncoder using json encoding.
type JSONClientEncoder struct{}

// JSONRequest is request of a RPC call encoded by json.
type JSONRequest struct {
	Param    json.RawMessage   `json:"param"`
	Timeout  string            `json:"timeout,omitempty"`
	Passthru map[string]string `json:"passthru,omitempty"`
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
	// Decode request.
	r := &JSONRequest{}
	if err := json.Unmarshal(data, r); err != nil {
		return err
	}

	// Decode param.
	reader := bytes.NewReader(r.Param)
	if err := jsonUnmarshaler.Unmarshal(reader, req.Param); err != nil {
		return err
	}

	// Optional passthru.
	req.Passthru = r.Passthru

	// Optional timeout.
	if r.Timeout != "" {
		dur, err := time.ParseDuration(r.Timeout)
		if err != nil {
			return err
		}
		req.Timeout = &dur
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
		// Encode result.
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

	// Optional passthru.
	r.Passthru = req.Passthru

	// Optional timeout.
	if req.Timeout != nil {
		r.Timeout = req.Timeout.String()
	}

	// Encode request.
	return json.Marshal(r)
}

// DecodeReply implements RPCClientEncoder interface.
func (e JSONClientEncoder) DecodeReply(data []byte, reply *RPCReply) error {

	// Decode reply.
	r := &JSONReply{}
	if err := json.Unmarshal(data, r); err != nil {
		return err
	}

	if r.Error != "" {
		// Error case.
		reply.Error = errors.New(r.Error)
		reply.Result = nil
		return nil
	} else {
		// Normal case. Decode result.
		reader := bytes.NewReader([]byte(r.Result))
		return jsonUnmarshaler.Unmarshal(reader, reply.Result)
	}

}
