package net

import (
	"encoding/json"

	"github.com/flarco/gutil"
	"github.com/spf13/cast"
)

// MessageType is an enum type for messages
type MessageType string

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
var NoReplyMsg = Message{Type: NoReplyMsgType}

// AckMsg is for handlers were all went well
var AckMsg = Message{Type: AckMsgType}

// JSON returns a JSON string
func (msg *Message) JSON() []byte {
	jBytes, _ := json.Marshal(msg)
	return jBytes
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

	return Message{
		ReqID:     gutil.NewTsID("msg"),
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
		ReqID:     gutil.NewTsID("msg"),
		Type:      msgType,
		Data:      gutil.M("payload", payload),
		OrigReqID: OrigReqID,
	}
}

// NewMessageErr create a new message with ErrMsg type
func NewMessageErr(err error, orgReqID ...string) Message {
	msg := NewMessage(ErrMsgType, nil, orgReqID...)
	msg.Error = err.Error()
	return msg
}

// NewMessageFromJSON creates a new message from a json
func NewMessageFromJSON(body []byte) (m Message, err error) {
	m = Message{}
	err = json.Unmarshal(body, &m)
	if err != nil {
		err = gutil.Error(err, "could not unmarshal message")
	}
	return
}
