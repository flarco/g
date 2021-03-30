package cacheredis

import (
	"context"
	"sync"
	"time"

	"github.com/flarco/g"
	"github.com/flarco/g/net"
	"github.com/go-redis/redis/v8"
)

// PubSub is a redis publish/subscription object
type PubSub struct {
	Name          string
	PubSub        *redis.PubSub
	Handlers      net.Handlers
	ReplyHandlers map[string]net.Handler
	mux           sync.Mutex
	c             *Cache
}

// AddHandler adds a new handler
func (ps *PubSub) AddHandler(key net.MessageType, h net.Handler) {
	ps.mux.Lock()
	ps.Handlers[key] = h
	ps.mux.Unlock()
}

// Publish publishes a message
func (ps *PubSub) Publish(topic string, msg net.Message) (err error) {
	msg.From = ps.Name
	return ps.c.Publish(topic, msg)
}

// PublishWait publishes a message and waits
func (ps *PubSub) PublishWait(topic string, msg net.Message, timeOut ...int) (rMsg net.Message, err error) {
	msg.From = ps.Name
	err = ps.c.Publish(topic, msg)
	if err != nil {
		err = g.Error(err, "could not publish to %s", topic)
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

	ps.mux.Lock()
	ps.ReplyHandlers[msg.ReqID] = replyHandler
	ps.mux.Unlock()

	// wait for response with timeout
	timer := time.NewTimer(to)
	select {
	case <-timer.C:
		err = g.Error("timeout. no response received for message %s", msg.Type)
		return
	case rMsg = <-replyChn:
		return
	}
}

// PublishContext publishes a message with context
func (ps *PubSub) PublishContext(ctx context.Context, topic string, msg net.Message) error {
	msg.From = ps.Name
	return ps.c.PublishContext(ctx, topic, msg)
}

// Loop processes messages and wait for reception
func (ps *PubSub) Loop() {
	for rcv := range ps.PubSub.Channel() {
		msg, err := net.NewMessageFromJSON([]byte(rcv.Payload))
		g.LogError(err, "could not parse received message @ "+ps.Name)
		if err == nil {
			go ps.HandleMsg(msg)
		}
	}
}

// HandleMsg handles a received message
func (ps *PubSub) HandleMsg(msg net.Message) {
	ps.mux.Lock()
	handler, ok := ps.Handlers[msg.Type]
	if !ok {
		handler, ok = ps.ReplyHandlers[msg.OrigReqID]
		delete(ps.ReplyHandlers, msg.OrigReqID)
	}
	ps.mux.Unlock()
	if ok {
		rMsg := handler(msg)
		rMsg.OrigReqID = msg.ReqID
		if rMsg.Type != "" && rMsg.Type != net.NoReplyMsgType {
			ps.Publish(msg.From, rMsg)
		}
	} else if msg.Type != net.AckMsgType {
		g.Warn("no handler found for msg type: %s", msg.Type)
	}
	return
}
