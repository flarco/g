package cachepg

import (
	"os"
	"testing"
	"time"

	"github.com/flarco/g"
	"github.com/flarco/g/net"
	"github.com/stretchr/testify/assert"
)

var (
	dbURL = os.Getenv("POSTGRES_URL")
)

func TestPubSub(t *testing.T) {
	c, err := NewCachePG(dbURL)
	assert.NoError(t, err)
	payload := "1234567"

	received := false
	handlers := HandlerMap{
		net.MessageType("test"): func(msg net.Message) (rMsg net.Message) {
			assert.Equal(t, msg.Payload(), payload)
			received = true
			return
		},
	}
	_, err = c.Subscribe("test_chan", handlers)
	assert.NoError(t, err)

	msg := net.NewMessagePayload(
		net.MessageType("test"),
		payload,
	)
	err = c.Publish("test_chan", msg)
	assert.NoError(t, err)
	time.Sleep(300 * time.Millisecond)
	assert.True(t, received)

	msg = net.NewMessage(
		net.MessageType("ping"),
		g.M("test", "ing"),
	)

	handlers = HandlerMap{
		net.MessageType("ping"): func(msg net.Message) (rMsg net.Message) {
			return net.NewMessage(
				net.MessageType("pong"),
				g.M("test", "received"),
			)
		},
	}
	_, err = c.Subscribe("test_chan2", handlers)
	assert.NoError(t, err)

	rMsg, err := c.PublishWait("test_chan2", msg, 2)
	assert.NoError(t, err)

	assert.EqualValues(t, "pong", rMsg.Type)
	assert.Equal(t, "received", rMsg.Data["test"])
}

func TestLock(t *testing.T) {
	var lockCnt int
	c, err := NewCachePG(dbURL)
	assert.NoError(t, err)

	tx1, err := c.Lock(LockType(1))
	assert.NoError(t, err)

	tx2, ok := c.LockTry(LockType(2))
	assert.True(t, ok)
	c.db.Get(&lockCnt, "SELECT count(1) from pg_locks where locktype = 'advisory'")
	assert.Equal(t, 2, lockCnt)

	assert.True(t, c.Unlock(tx1, LockType(1)))
	assert.True(t, c.Unlock(tx2, LockType(2)))

	c.db.Get(&lockCnt, "SELECT count(1) from pg_locks where locktype = 'advisory'")
	assert.Equal(t, 0, lockCnt)

}

func TestSetGet(t *testing.T) {
	c, err := NewCachePG(dbURL)
	assert.NoError(t, err)

	err = c.createTable()
	if !assert.NoError(t, err) {
		return
	}
	defer c.dropTable()

	err = c.DeleteExpired()
	assert.NoError(t, err)

	err = c.Set("key-1", "a stupid error")
	assert.NoError(t, err)

	arr := []interface{}{1, "a", true}
	m := g.M("a", arr)
	err = c.SetM("key-2", g.M("nested", m, "arr", arr))
	assert.NoError(t, err)

	keys, err := c.GetLikeKeys("key-%")
	assert.Len(t, keys, 2)

	vals, err := c.GetLikeValuesM("key-%")
	assert.Len(t, vals, 2)

	val, err := c.Get("key-1")
	assert.Equal(t, "a stupid error", val)

	found, err := c.Has("key-1")
	assert.NoError(t, err)
	assert.True(t, found)

	err = c.SetEx("key-1", "another stupid error", 1)
	assert.NoError(t, err)

	val, err = c.Get("key-1")
	assert.Equal(t, "another stupid error", val)

	time.Sleep(2 * time.Second)

	// should not be there since expired
	val, err = c.Get("key-1")
	assert.NoError(t, err)
	assert.Nil(t, val)

	found, err = c.Has("key-1")
	assert.NoError(t, err)
	assert.False(t, found)

	valM, err := c.Pop("key-2")
	assert.NoError(t, err)

	// should not be there since popped
	valM, err = c.GetM("key-2")
	assert.NoError(t, err)
	assert.Nil(t, valM)
}

func BenchmarkCachePGSet(b *testing.B) {
	c, err := NewCachePG(dbURL)
	g.LogFatal(err)
	for n := 0; n < b.N; n++ {
		c.Set("key-1", "a stupid error")
	}
}

func BenchmarkCachePGGet(b *testing.B) {
	c, err := NewCachePG(dbURL)
	g.LogFatal(err)
	c.Set("key-1", "a stupid error")
	for n := 0; n < b.N; n++ {
		c.Get("key-1")
	}
}

func BenchmarkCachePGLock(b *testing.B) {
	c, err := NewCachePG(dbURL)
	g.LogFatal(err)
	for n := 0; n < b.N; n++ {
		tx, _ := c.Lock(LockType(1))
		c.Unlock(tx, LockType(1))
	}
}

func BenchmarkCachePGPubSub(b *testing.B) {
	c, err := NewCachePG(dbURL)
	g.LogFatal(err)

	handlers := HandlerMap{
		net.MessageType("test"): func(msg net.Message) (rMsg net.Message) {
			return
		},
	}

	payload := "1234567"
	msg := net.NewMessagePayload(
		net.MessageType("test"),
		payload,
	)
	_, err = c.Subscribe("test_chan", handlers)
	for n := 0; n < b.N; n++ {
		err = c.Publish("test_chan", msg)
		g.LogFatal(err)
	}
}
