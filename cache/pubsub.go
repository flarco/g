package cache

import (
	"fmt"
	"time"

	g "github.com/flarco/gutil"
	"github.com/flarco/gutil/net"
	"github.com/lib/pq"
	"github.com/spf13/cast"
)

type funcMap map[net.MessageType]func(msg net.Message) (rMsg net.Message)
type replyMap map[string]func(msg net.Message) (rMsg net.Message)

// Listener a PG listener / subscription
type Listener struct {
	Context  g.Context
	Channel  string
	listener *pq.Listener
	callback func(payload string)
}

// Close closes the listener connection
func (l *Listener) Close() {
	l.listener.Close()
}

// ListenLoop is the loop process of a listener to receive a message
func (l *Listener) ListenLoop() {
	defer l.listener.Close()
	for {
		select {
		case <-l.Context.Ctx.Done():
			return
		case n := <-l.listener.Notify:
			l.callback(n.Extra)
		case <-time.After(90 * time.Second):
			err := l.listener.Ping()
			if err != nil {
				g.LogError(err, "no listener connection")
				return
			}
		}
	}
}

// subscribeDefault subs to a default channel
func (c *Cache) subscribeDefault() {
	handler := func(payload string) {
		msg, err := net.NewMessageFromJSON([]byte(payload))
		if err != nil {
			err = g.Error(err, "could not parse message from JSON")
		} else {
			_, err = c.cacheHandler(msg)
		}
		g.LogError(err)
	}
	c.defChannel = g.RandString(g.AlphaRunes, 7)
	c.Subscribe(c.defChannel, handler)
}

// cacheHandler handles incoming messages
func (c *Cache) cacheHandler(msg net.Message) (rMsg net.Message, err error) {
	srcChannel := cast.ToString(msg.Data["src_channel"])
	rMsg = c.handleMsg(msg)
	rMsg.OrigReqID = msg.ReqID
	if rMsg.Type != net.NoReplyMsgType && srcChannel != c.defChannel {
		err = c.PublishMsg(srcChannel, rMsg)
	} else if rMsg.IsError() {
		err = g.Error(rMsg.Error)
	}
	return
}

func (c *Cache) handleMsg(msg net.Message) (rMsg net.Message) {
	rMsg = net.NoReplyMsg

	c.mux.Lock()
	handler, ok := c.replyHandlers[msg.OrigReqID]
	if ok {
		delete(c.replyHandlers, msg.OrigReqID)
	} else {
		handler, ok = c.handlers[msg.Type]
	}
	c.mux.Unlock()

	if ok {
		rMsg = handler(msg)
	} else {
		err := g.Error(g.F("no cache handler for %s", msg.Type))
		rMsg = net.NewMessageErr(err)
	}

	return
}

// AddHandler adds a handler for an incoming message type
func (c *Cache) AddHandler(msgType net.MessageType, handler func(msg net.Message) (rMsg net.Message)) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.handlers[msgType] = handler
}

// AddReplyHandler adds a handler for an incoming reply
func (c *Cache) AddReplyHandler(reqID string, handler func(msg net.Message) (rMsg net.Message), timeout time.Duration) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.replyHandlers[reqID] = handler

	// delete reply handler after timer
	time.AfterFunc(
		timeout,
		func() {
			c.mux.Lock()
			delete(c.replyHandlers, reqID)
			c.mux.Unlock()
		},
	)
}

// Subscribe to a PG notification channel
func (c *Cache) Subscribe(channel string, callback func(p string)) (l *Listener, err error) {
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
	l = &Listener{g.NewContext(c.Context.Ctx), channel, listener, callback}
	c.listeners.Set(channel, l)
	go l.ListenLoop()

	return
}

// Publish to a PG notification channel
func (c *Cache) Publish(channel string, payload string) (err error) {
	_, err = c.db.ExecContext(c.Context.Ctx, "SELECT pg_notify($1, $2)", channel, payload)
	if err != nil {
		err = g.Error(err, "unable to publish payload to "+channel)
	}
	return
}

// PublishMsg a message to a PG notification channel
func (c *Cache) PublishMsg(channel string, msg net.Message) (err error) {
	msg.Data["src_channel"] = c.defChannel
	err = c.Publish(channel, string(msg.JSON()))
	if err != nil {
		err = g.Error(err, "unable to publish msg to "+channel)
	}
	return
}

// PublishMsgWait publishes a msg to a PG notification channel and waits for a reply
// default timeout is 10 seconds.
func (c *Cache) PublishMsgWait(channel string, msg net.Message, timeOut ...int) (rMsg net.Message, err error) {
	if channel == c.defChannel {
		rMsg = msg
		return
	}

	to := 10 * time.Second
	if len(timeOut) > 0 {
		to = time.Duration(timeOut[0]) * time.Second
	}

	replyChn := make(chan net.Message)
	replyHandler := func(msg net.Message) net.Message {
		replyChn <- msg
		return net.AckMsg
	}

	c.AddReplyHandler(msg.ReqID, replyHandler, to)
	err = c.PublishMsg(channel, msg)
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
