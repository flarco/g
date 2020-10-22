package cachepg

import (
	"fmt"
	"sync"
	"time"

	g "github.com/flarco/gutil"
	"github.com/flarco/gutil/net"
	"github.com/lib/pq"
	"github.com/spf13/cast"
)

type (
	// HandlerFunc is a function for a handler
	HandlerFunc func(msg net.Message) (rMsg net.Message)
	// HandlerMap is a map of handler functions for newly received messages
	HandlerMap map[net.MessageType]HandlerFunc
	// ReplyMap is a map of handler functions for replied messages
	ReplyMap map[string]HandlerFunc
)

// Listener a PG listener / subscription
type Listener struct {
	Context       g.Context
	Channel       string
	mux           sync.Mutex
	listener      *pq.Listener
	handlers      HandlerMap
	replyHandlers ReplyMap
	c             *Cache
}

// Close closes the listener connection
func (l *Listener) Close() {
	l.listener.Close()
}

// ProcessMsg processes a received message
func (l *Listener) ProcessMsg(msg net.Message) (rMsg net.Message) {
	var err error

	// g.Debug("Listener Channel: " + l.Channel)

	l.mux.Lock()
	handler, ok := l.replyHandlers[msg.OrigReqID]
	if ok {
		delete(l.replyHandlers, msg.OrigReqID)
	} else {
		handler, ok = l.handlers[msg.Type]
	}
	l.mux.Unlock()

	if ok {
		rMsg = handler(msg)
		if rMsg.Type == net.MessageType("") {
			rMsg = net.NoReplyMsg
		}
	} else {
		err = g.Error(g.F("no handler for %s - listener %s", msg.Type, l.Channel))
		// rMsg = net.NewMessageErr(err)
	}

	toChannel := cast.ToString(rMsg.Data["to_channel"])
	if toChannel != "" {
		delete(rMsg.Data, "to_channel")
	} else {
		toChannel = cast.ToString(msg.Data["from_channel"])
	}

	rMsg.OrigReqID = msg.ReqID
	// g.P(msg)
	// g.P(rMsg)
	if rMsg.Type != net.NoReplyMsgType && toChannel != l.Channel {
		err = g.ErrorIf(l.c.Publish(toChannel, rMsg))
	} else if rMsg.IsError() {
		err = g.Error(rMsg.Error)
	}
	g.LogError(err, "error processing msg")
	return
}

// ListenLoop is the loop process of a listener to receive a message
func (l *Listener) ListenLoop() {
	defer l.listener.Close()
	for {
		select {
		case <-l.Context.Ctx.Done():
			return
		case n := <-l.listener.Notify:
			if n == nil {
				return
			}
			msg, err := net.NewMessageFromJSON([]byte(n.Extra))
			g.LogError(err, "error parsing msg")
			if err == nil {
				l.ProcessMsg(msg)
			}
		case <-time.After(90 * time.Second):
			err := l.listener.Ping()
			if err != nil {
				g.LogError(err, "no listener connection")
				return
			}
		}
	}
}

// AddHandlers adds handlers for incoming message types
func (l *Listener) AddHandlers(handlers HandlerMap) {
	l.mux.Lock()
	defer l.mux.Unlock()
	for msgType, handler := range handlers {
		l.handlers[msgType] = handler
	}
}

// AddReplyHandler adds a handler for an incoming reply
func (l *Listener) AddReplyHandler(reqID string, handler HandlerFunc, timeout time.Duration) {
	l.mux.Lock()
	defer l.mux.Unlock()
	l.replyHandlers[reqID] = handler

	// delete reply handler after timer
	time.AfterFunc(
		timeout,
		func() {
			l.mux.Lock()
			delete(l.replyHandlers, reqID)
			l.mux.Unlock()
		},
	)
}

// subscribeDefault subs to a default channel
func (c *Cache) subscribeDefault(chanName string) {
	c.defChannel = chanName
	c.Subscribe(c.defChannel, HandlerMap{})
}

// DefListener returns the default listener instance
func (c *Cache) DefListener() (l *Listener) {
	if lI, ok := c.listeners.Get(c.defChannel); ok {
		l = lI.(*Listener)
	}
	return
}

// Subscribe to a PG notification channel
func (c *Cache) Subscribe(channel string, handlers HandlerMap) (l *Listener, err error) {
	logEventErr := func(ev pq.ListenerEventType, err error) {
		mapping := map[pq.ListenerEventType]string{
			pq.ListenerEventConnected:               "ListenerEventConnected",
			pq.ListenerEventDisconnected:            "ListenerEventDisconnected",
			pq.ListenerEventReconnected:             "ListenerEventReconnected",
			pq.ListenerEventConnectionAttemptFailed: "ListenerEventConnectionAttemptFailed",
		}
		g.LogError(err, "message from %s: %s", channel, mapping[ev])
	}

	listener := pq.NewListener(c.dbURL, 50*time.Millisecond, time.Minute, logEventErr)
	err = listener.Listen(channel)
	if err != nil {
		err = g.Error(err, "could not listen to channel "+channel)
		return
	}

	if lI, ok := c.listeners.Get(channel); ok {
		lI.(*Listener).Context.Cancel()
		c.listeners.Remove(channel)
	}
	l = &Listener{
		Context:       g.NewContext(c.Context.Ctx),
		Channel:       channel,
		listener:      listener,
		handlers:      handlers,
		replyHandlers: ReplyMap{},
		c:             c,
	}
	c.listeners.Set(channel, l)
	go l.ListenLoop()

	return
}

// publish to a PG notification channel
func (c *Cache) publish(channel string, payload string) (err error) {
	_, err = c.publishStmt.ExecContext(c.Context.Ctx, channel, payload)
	if err != nil {
		err = g.Error(err, "unable to publish payload to "+channel)
	}
	return
}

// Publish a message to a PG notification channel
func (c *Cache) Publish(channel string, msg net.Message) (err error) {
	if channel == "" {
		return g.Error("empty channel provided")
	} else if msg.Type == net.MessageType("") {
		return nil
	}
	msg.Data["from_channel"] = c.defChannel
	err = c.publish(channel, string(msg.JSON()))
	if err != nil {
		err = g.Error(err, "unable to publish msg to "+channel)
	}
	return
}

// PublishWait publishes a msg to a PG notification channel and waits for a reply
// default timeout is 10 seconds.
func (c *Cache) PublishWait(channel string, msg net.Message, timeOut ...int) (rMsg net.Message, err error) {
	if channel == c.defChannel {
		msg.Data["from_channel"] = c.defChannel
		rMsg = c.DefListener().ProcessMsg(msg)
		return
	}

	to := 10 * time.Second
	if len(timeOut) > 0 {
		to = time.Duration(timeOut[0]) * time.Second
	}

	replyChn := make(chan net.Message)
	replyHandler := func(msg net.Message) net.Message {
		replyChn <- msg
		return net.NoReplyMsg
	}

	c.DefListener().AddReplyHandler(msg.ReqID, replyHandler, to)
	err = c.Publish(channel, msg)
	if err != nil {
		err = g.Error(err, "could not publish to %s", channel)
		return
	}

	// wait for response with timeout
	timer := time.NewTimer(to)
	select {
	case <-timer.C:
		err = g.Error(fmt.Errorf("timeout. no response received for message %s", msg.Type))
		return
	case rMsg = <-replyChn:
		return
	}
}
