package net

import (
	"encoding/json"
	"github.com/flarco/gutil"
	"github.com/spf13/cast"
)

const (
	// NoReply is to not reply
	NoReply MessageType = "no_reply"

	// MsgReceivedOK is an acknowledgment of receipt
	MsgReceivedOK MessageType = "receipt_ok"

	// ErrMsg is an error message
	ErrMsg MessageType = "error"

	// ClientHeartbeatMsg is for client heartbeat messages
	ClientHeartbeatMsg MessageType = "client_heartbeat"

	// LogEventMsg is for arbitrary events
	LogEventMsg MessageType = "log_event"

	// LogStreamReqMsg is a request for log events stream
	LogStreamReqMsg MessageType = "log_stream_request"

)

const (
	MsgScopeAccount   MessageScopeLevel = "account"
	MsgScopeExecution MessageScopeLevel = "execution"
	MsgScopeJob       MessageScopeLevel = "job"
	MsgScopeWorker    MessageScopeLevel = "worker"
	MsgScopeAll       MessageScopeLevel = "all"
)

// MessageType is an enum type for messages
type MessageType string

func (mt MessageType) String() string {
	return string(mt)
}

// NoReplyMsg is for handlers who don't reply messages
var NoReplyMsg = Message{Type: NoReply}

// Message is a basic protocol for communcation
type Message struct {
	ReqID     string                 `json:"req_id"`
	Type      MessageType            `json:"type"`
	Source    EventSource            `json:"source,omitempty"`
	Scope     MessageScope           `json:"scope,omitempty"`
	Payload   string                 `json:"payload,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error"`
	OrigReqID string                 `json:"orig_req_id,omitempty"`

	// HTTP proxying
	Code   int    `json:"code,omitempty"`
	Method string `json:"method,omitempty"`
	Route  string `json:"route,omitempty"`
}

func (msg *Message) String() string {
	return gutil.F("%s: %s", msg.Type, string(msg.Payload))
}

// Text returns the text value and deletes it from data map
func (msg *Message) Text() string {
	text := cast.ToString(msg.Data["text"])
	delete(msg.Data, "text")
	return text
}

// MessageScope is the scope of the message
type MessageScope struct {
	Level     MessageScopeLevel `json:"level"`
	AccountID int               `json:"account_id"`
	UserID    int               `json:"user_id"`
	ExecID    int               `json:"exec_id"`
	JobID     int               `json:"job_id"`
	JobName   string            `json:"job_name"`
}

// MessageScopeLevel is the scope level
type MessageScopeLevel string

func (ms MessageScopeLevel) String() string {
	return string(ms)
}

// Console returns a timestamped string
func (msg *Message) Console() string {
	text := cast.ToString(msg.Data["text"])
	return gutil.F("%s -- %s | %s", gutil.TimeColored(), string(msg.Source), text)
}

// JSON returns a JSON string
func (msg *Message) JSON() []byte {
	jBytes, _ := json.Marshal(msg)
	return jBytes
}

// ToMap returns a map of interface
func (msg *Message) ToMap() map[string]interface{} {
	m := gutil.M()
	for k, v := range msg.Data {
		m[k] = v // need to copy vals to avoid fatal error: concurrent map iteration and map write
	}
	return m
}

// NewMessageScope creates a new MessageScope
func NewMessageScope(level MessageScopeLevel) MessageScope {
	return MessageScope{
		Level: level,
	}
}

// NewMessage creates a new message with a payload
func NewMessage(msgType MessageType, msgPayload []byte, orgReqID ...string) Message {
	OrigReqID := ""
	if len(orgReqID) > 0 {
		OrigReqID = orgReqID[0]
	}

	return Message{
		ReqID:     gutil.NewTsID("msg"),
		Type:      msgType,
		Payload:   string(msgPayload),
		OrigReqID: OrigReqID,
		Data:      map[string]interface{}{},
	}
}

// NewMessage2 creates a new message with a map
func NewMessage2(msgType MessageType, data map[string]interface{}, orgReqID ...string) Message {
	OrigReqID := ""
	if len(orgReqID) > 0 {
		OrigReqID = orgReqID[0]
	}

	return Message{
		ReqID:     gutil.NewTsID(),
		Type:      msgType,
		Data:      data,
		OrigReqID: OrigReqID,
	}
}

// NewEventMessage creates a new event message
func NewEventMessage(data map[string]interface{}) Message {
	return Message{
		ReqID: gutil.NewTsID(),
		Type:  LogEventMsg,
		Data:  data,
	}
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

// NewMessageErr create a new message with ErrMsg type
func NewMessageErr(err error) Message {
	msg := NewMessage(ErrMsg, []byte(gutil.ErrMsgSimple(err)))
	msg.Error = gutil.ErrMsgSimple(err)
	return msg
}
