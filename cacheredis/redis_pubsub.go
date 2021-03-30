package cacheredis

import (
	"context"
	"sync"

	"github.com/flarco/g"
	"github.com/flarco/g/net"
	"github.com/go-redis/redis/v8"
)

// PubSub is a redis publish/subscription object
type PubSub struct {
	Name     string
	PubSub   *redis.PubSub
	Handlers net.Handlers
	mux      sync.Mutex
	c        *Cache
}

// Publish publishes a message
func (ps *PubSub) Publish(topic string, msg net.Message) (err error) {
	msg.From = ps.Name
	return ps.c.Publish(topic, msg)
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
	ps.mux.Unlock()
	if ok {
		rMsg := handler(msg)
		rMsg.OrigReqID = msg.ReqID
		if rMsg.Type != "" && rMsg.Type != net.NoReplyMsgType {
			ps.Publish(msg.From, rMsg)
		}
	} else {
		g.Warn("no handler found for msg type: %s", msg.Type)
	}
	return
}
