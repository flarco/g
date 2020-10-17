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

// PublishWait to a PG notification channel and waits for a reply
func (c *Cache) PublishWait(channel string, payload string) (rPayload string,
	err error) {
	err = c.Publish(channel, payload)
	if err != nil {
		err = g.Error(err, "unable to publish payload to "+channel)
	}
	// wait for reply
	// TODO:

	return
}
