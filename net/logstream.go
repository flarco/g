package net

import (
	"encoding/json"
	"github.com/flarco/gutil"
	"sync"
	"time"
)

const (
	// EventSourceMaster is master generated event
	EventSourceMaster EventSource = "master"
	// EventSourceWorker is worker generated event
	EventSourceWorker EventSource = "worker"
	// EventSourceTask is task generated event
	EventSourceTask EventSource = "task"
)

const (
	// LogEventNotifyToast is to notify UI with a toast message
	LogEventNotifyToast LogEventType = "notify_toast"
)

// LogStreamRequest is a request for a log stream
type LogStreamRequest struct {
	ID    string       `json:"id"`
	Scope MessageScope `json:"scope"`
}

// JSON returns a JSON string
func (lsr *LogStreamRequest) JSON() []byte {
	jBytes, _ := json.Marshal(lsr)
	return jBytes
}

// ToMap returns a map
func (lsr *LogStreamRequest) ToMap() map[string]interface{} {
	m := gutil.M(
		"id", lsr.ID,
		"scope", lsr.Scope,
		"account_id", lsr.Scope.AccountID,
		"user_id", lsr.Scope.UserID,
		"exec_id", lsr.Scope.ExecID,
		"job_id", lsr.Scope.JobID,
		"job_name", lsr.Scope.JobName,
	)
	return m
}

// EventSource is the source of the event
type EventSource string

// LogEventType is the type of event
type LogEventType string

// LogStream is a pub/sub log stream
type LogStream struct {
	Channels    map[string]*LogStreamChan
	ExecSubs    map[int]map[string]*LogStreamChan
	JobSubs     map[int]map[string]*LogStreamChan
	AccountSubs map[int]map[string]*LogStreamChan
	AllSubs     map[string]*LogStreamChan
	Closed      bool
	mu          sync.Mutex
}

// LogStreamChan is a single stream
type LogStreamChan struct {
	Chn         chan Message
	ConnectTime time.Time
	lsr         *LogStreamRequest
	closed      bool
	mu          sync.Mutex
}

// Lsr returns the channel's LogStreamRequest
func (lsc *LogStreamChan) Lsr() *LogStreamRequest {
	return lsc.lsr
}

// close closes the log stream channel
func (lsc *LogStreamChan) close() {
	lsc.mu.Lock()
	defer lsc.mu.Unlock()
	if !lsc.closed {
		close(lsc.Chn)
	}
	lsc.closed = true
}

// Push pushes a string to the channel
func (lsc *LogStreamChan) Push(s string) {
	msg := NewEventMessage(gutil.M("text", s))
	lsc.PushEvent(msg)
}

// PushEvent pushes a message to the channel
func (lsc *LogStreamChan) PushEvent(msg Message) {
	lsc.mu.Lock()
	defer lsc.mu.Unlock()
	if !lsc.closed {
		lsc.Chn <- msg
	}
}

// UnsubscribeID unsubscribes and closes the provided channel id
func (ls *LogStream) UnsubscribeID(id string) {
	lsc, ok := ls.Channels[id]
	if ok {
		ls.Unsubscribe(lsc)
	}
}

// Unsubscribe unsubscribes and closes the provided channel
func (ls *LogStream) Unsubscribe(lsc *LogStreamChan) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	lsr := lsc.lsr
	switch lsr.Scope.Level {
	case MsgScopeExecution:
		delete(ls.ExecSubs[lsr.Scope.ExecID], lsr.ID)
		if len(ls.ExecSubs[lsr.Scope.ExecID]) == 0 {
			delete(ls.ExecSubs, lsr.Scope.ExecID)
		}
	case MsgScopeJob:
		delete(ls.JobSubs[lsr.Scope.JobID], lsr.ID)
		if len(ls.JobSubs[lsr.Scope.JobID]) == 0 {
			delete(ls.JobSubs, lsr.Scope.JobID)
		}
	case MsgScopeAccount:
		delete(ls.AccountSubs[lsr.Scope.AccountID], lsr.ID)
		if len(ls.AccountSubs[lsr.Scope.AccountID]) == 0 {
			delete(ls.AccountSubs, lsr.Scope.AccountID)
		}
	case MsgScopeAll:
		delete(ls.AllSubs, lsr.ID)
	}
	delete(ls.Channels, lsr.ID)

	lsc.close()
	gutil.Debug("Unsubscribed Log Stream: " + lsr.ID)
}

// Subscribe subscribes to a log stream
func (ls *LogStream) Subscribe(lsr *LogStreamRequest) (lsc *LogStreamChan) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	gutil.Debug("Subscribed Log Stream: " + lsr.ID)
	lsc = newLogStreamChan(lsr)
	ls.Channels[lsr.ID] = lsc
	switch lsr.Scope.Level {
	case MsgScopeExecution:
		if _, ok := ls.ExecSubs[lsr.Scope.ExecID]; !ok {
			ls.ExecSubs[lsr.Scope.ExecID] = map[string]*LogStreamChan{}
		}
		ls.ExecSubs[lsr.Scope.ExecID][lsr.ID] = lsc
	case MsgScopeJob:
		if _, ok := ls.JobSubs[lsr.Scope.JobID]; !ok {
			ls.JobSubs[lsr.Scope.JobID] = map[string]*LogStreamChan{}
		}
		ls.JobSubs[lsr.Scope.JobID][lsr.ID] = lsc
	case MsgScopeAccount:
		if _, ok := ls.AccountSubs[lsr.Scope.AccountID]; !ok {
			ls.AccountSubs[lsr.Scope.AccountID] = map[string]*LogStreamChan{}
		}
		ls.AccountSubs[lsr.Scope.AccountID][lsr.ID] = lsc
	case MsgScopeAll:
		ls.AllSubs[lsr.ID] = lsc
	}
	return
}

// Publish publishes event to subs
func (ls *LogStream) Publish(msg Message) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	for _, lsc := range ls.AllSubs {
		go lsc.PushEvent(msg)
	}

	switch msg.Scope.Level {
	case MsgScopeExecution:
		if subs, ok := ls.ExecSubs[msg.Scope.ExecID]; ok {
			for _, lsc := range subs {
				go lsc.PushEvent(msg)
			}
		}
		if subs, ok := ls.JobSubs[msg.Scope.JobID]; ok {
			for _, lsc := range subs {
				go lsc.PushEvent(msg)
			}
		}
		if subs, ok := ls.AccountSubs[msg.Scope.AccountID]; ok {
			for _, lsc := range subs {
				go lsc.PushEvent(msg)
			}
		}
	case MsgScopeJob:
		if subs, ok := ls.JobSubs[msg.Scope.JobID]; ok {
			for _, lsc := range subs {
				go lsc.PushEvent(msg)
			}
		}
		if subs, ok := ls.AccountSubs[msg.Scope.AccountID]; ok {
			for _, lsc := range subs {
				go lsc.PushEvent(msg)
			}
		}
	case MsgScopeAccount:
		if subs, ok := ls.AccountSubs[msg.Scope.AccountID]; ok {
			for _, lsc := range subs {
				go lsc.PushEvent(msg)
			}
		}
	}
}

// NewLogStreamRequest returns new NewLogStreamRequest
func NewLogStreamRequest(scopeLevel MessageScopeLevel) LogStreamRequest {
	return LogStreamRequest{
		ID:    gutil.NewTsID("lsr"),
		Scope: NewMessageScope(scopeLevel),
	}
}

// NewLogStreamRequestFromMsg returns new NewLogStreamRequest from message
func NewLogStreamRequestFromMsg(msg Message) *LogStreamRequest {
	lsr := LogStreamRequest{
		ID:    msg.ReqID,
		Scope: msg.Scope,
	}
	return &lsr
}

// NewLogStream creates a client stream
func NewLogStream() *LogStream {
	LogStream := LogStream{
		Channels:    map[string]*LogStreamChan{},
		ExecSubs:    map[int]map[string]*LogStreamChan{},
		JobSubs:     map[int]map[string]*LogStreamChan{},
		AccountSubs: map[int]map[string]*LogStreamChan{},
		AllSubs:     map[string]*LogStreamChan{},
	}
	return &LogStream
}

func newLogStreamChan(lsr *LogStreamRequest) *LogStreamChan {
	lsc := LogStreamChan{lsr: lsr, Chn: make(chan Message), ConnectTime: time.Now()}
	return &lsc
}
