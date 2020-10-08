package cache

import (
	"time"

	g "github.com/flarco/gutil"
	"github.com/lib/pq"
)

// Listener a PG listener / subscription
type Listener struct {
	Context  g.Context
	Channel  string
	listener *pq.Listener
	callback func(n *pq.Notification)
}

// ListenLoop is the loop process of a listener to receive a message
func (l *Listener) ListenLoop() {
	for {
		select {
		case <-l.Context.Ctx.Done():
			l.listener.Close()
			return
		case n := <-l.listener.Notify:
			l.callback(n)
		}
	}
}

// Subscribe to a PG notification channel
func (c *Cache) Subscribe(channel string, callback func(n *pq.Notification)) {
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
	err := listener.Listen(channel)
	g.LogError(err, "could not listen to channel "+channel)
	if err == nil {
		if lI, ok := c.listeners.Get(channel); ok {
			l := lI.(*Listener)
			l.listener.Close()
			l.Context.Cancel()
			c.listeners.Remove(channel)
		}
		l := &Listener{g.NewContext(c.Context.Ctx), channel, listener, callback}
		c.listeners.Set(channel, l)
		go l.ListenLoop()
	}
}
