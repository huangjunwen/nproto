package enc

import (
	"bytes"
	"encoding/json"

	"github.com/golang/protobuf/jsonpb"
)

// JSONPublisherEncoder is MsgPublisherEncoder using json encoding.
type JSONPublisherEncoder struct{}

// JSONSubscriberEncoder is MsgSubscriberEncoder using json encoding.
type JSONSubscriberEncoder struct{}

type JSONPayload struct {
	Msg      json.RawMessage   `json:"msg"`
	Passthru map[string]string `json:"passthru,omitempty"`
}

var (
	_ MsgPublisherEncoder  = JSONPublisherEncoder{}
	_ MsgSubscriberEncoder = JSONSubscriberEncoder{}
)

var (
	jsonUnmarshaler = jsonpb.Unmarshaler{
		AllowUnknownFields: true,
	}
	jsonMarshaler = jsonpb.Marshaler{
		EmitDefaults: true,
	}
)

// EncodePayload implements MsgPublisherEncoder interface.
func (e JSONPublisherEncoder) EncodePayload(payload *MsgPayload) ([]byte, error) {
	p := &JSONPayload{}

	// Encode msg.
	buf := &bytes.Buffer{}
	if err := jsonMarshaler.Marshal(buf, payload.Msg); err != nil {
		return nil, err
	}
	p.Msg = json.RawMessage(buf.Bytes())

	// Optional passthru.
	p.Passthru = payload.Passthru

	// Encode payload.
	return json.Marshal(p)
}

// DecodePayload implements MsgSubscriberEncoder interface.
func (e JSONSubscriberEncoder) DecodePayload(data []byte, payload *MsgPayload) error {
	// Decode payload.
	p := &JSONPayload{}
	if err := json.Unmarshal(data, p); err != nil {
		return err
	}

	// Decode msg.
	reader := bytes.NewReader(p.Msg)
	if err := jsonUnmarshaler.Unmarshal(reader, payload.Msg); err != nil {
		return err
	}

	// Optional passthru.
	payload.Passthru = p.Passthru
	return nil
}
