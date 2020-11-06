package net

import (
	"github.com/flarco/g"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
)

// MessageType is an enum type for messages
type MessageType string

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (mt MessageType) String() string {
	return string(mt)
}

const (
	// NoReplyMsgType is to not reply
	NoReplyMsgType MessageType = "no_reply"

	// AckMsgType is an acknowledgment of receipt
	AckMsgType MessageType = "ack"

	// ErrMsgType is an error message
	ErrMsgType MessageType = "error"
)

// Message is a basic protocol for communication
type Message struct {
	ReqID     string                 `json:"req_id"`
	Type      MessageType            `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error"`
	OrigReqID string                 `json:"orig_req_id,omitempty"`
}

// NoReplyMsg is for handlers who don't reply messages
var NoReplyMsg = Message{Type: NoReplyMsgType, Data: g.M()}

// AckMsg is for handlers where all went well
var AckMsg = Message{Type: AckMsgType, Data: g.M()}

// JSON returns a JSON string
func (msg *Message) JSON() []byte {
	jBytes, _ := json.Marshal(msg)
	return jBytes
}

// Text returns the text value and deletes it from data map
func (msg *Message) Text() string {
	text := cast.ToString(msg.Data["text"])
	delete(msg.Data, "text")
	return text
}

// Unmarshal parses a payload of JSON string into pointer
func (msg *Message) Unmarshal(objPtr interface{}) error {
	payload := cast.ToString(msg.Data["payload"])
	err := json.Unmarshal([]byte(payload), objPtr)
	if err != nil {
		err = g.Error(err, "could not unmarshal")
	}
	return err
}

// Payload returns the string payload
func (msg *Message) Payload() string {
	return cast.ToString(msg.Data["payload"])
}

// IsError returns true if an error message
func (msg *Message) IsError() bool {
	return msg.Type == ErrMsgType
}

// NewMessage creates a new message with a map
func NewMessage(msgType MessageType, data map[string]interface{}, orgReqID ...string) Message {
	OrigReqID := ""
	if len(orgReqID) > 0 {
		OrigReqID = orgReqID[0]
	}

	if data == nil {
		data = g.M()
	}

	return Message{
		ReqID:     g.NewTsID("msg"),
		Type:      msgType,
		Data:      data,
		OrigReqID: OrigReqID,
	}
}

// NewMessagePayload creates a new message with a map
func NewMessagePayload(msgType MessageType, payload string, orgReqID ...string) Message {
	OrigReqID := ""
	if len(orgReqID) > 0 {
		OrigReqID = orgReqID[0]
	}

	return Message{
		ReqID:     g.NewTsID("msg"),
		Type:      msgType,
		Data:      g.M("payload", payload),
		OrigReqID: OrigReqID,
	}
}

// NewMessageObj creates a new message with a obj
func NewMessageObj(msgType MessageType, obj interface{}, orgReqID ...string) Message {
	OrigReqID := ""
	if len(orgReqID) > 0 {
		OrigReqID = orgReqID[0]
	}

	return Message{
		ReqID:     g.NewTsID("msg"),
		Type:      msgType,
		Data:      g.M("payload", g.Marshal(obj)),
		OrigReqID: OrigReqID,
	}
}

// NewMessageErr create a new message with ErrMsg type
func NewMessageErr(err error, orgReqID ...string) Message {
	msg := NewMessage(ErrMsgType, nil, orgReqID...)

	E, ok := err.(*g.ErrType)
	if !ok {
		E = g.NewError(3, err).(*g.ErrType)
	}
	msg.Error = E.Full()
	msg.Data["error_debug"] = E.Debug()
	return msg
}

// NewMessageFromJSON creates a new message from a json
func NewMessageFromJSON(body []byte) (m Message, err error) {
	m = Message{}
	err = json.Unmarshal(body, &m)
	if err != nil {
		err = g.Error(err, "could not unmarshal message")
	}
	return
}
