package cacheredis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/flarco/g"
	"github.com/flarco/g/net"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	Debug = true
	c, err := NewCache(Config{
		URL: os.Getenv("REDIS_MPC_URL"),
		Ctx: context.Background(),
	})
	assert.NoError(t, err)

	key := "keytest"
	val := "value"
	err = c.Set(key, val)
	assert.NoError(t, err)

	val2 := ""
	err = c.Pop(key, &val2)
	assert.NoError(t, err)
	assert.Equal(t, val, val2)

	err = c.Get(key, &val2)
	assert.Error(t, err)

	err = c.SetEx(key, val, 1)

	time.Sleep(2 * time.Second)
	err = c.Get(key, &val2)
	assert.Error(t, err)

	m := g.M("k1", "val", "k2", "val2")
	err = c.HSetAll("testhash", m)
	assert.NoError(t, err)

	err = c.HSet("testhash", "k3", "val3")
	assert.NoError(t, err)

	found := c.HHas("testhash", "k1")
	assert.True(t, found)

	err = c.HGet("testhash", "k2", &val2)
	assert.NoError(t, err)
	assert.Equal(t, "val2", val2)

	err = c.HGet("testhash", "k3", &val2)
	assert.NoError(t, err)
	assert.Equal(t, "val3", val2)

	err = c.HGet("testhash", "k4", &val2)
	assert.Error(t, err)

	err = c.HDel("testhash", "k2")
	assert.NoError(t, err)

	err = c.HGet("testhash", "k2", &val2)
	assert.Error(t, err)

	m2, err := c.HGetAll("testhash")
	assert.NoError(t, err)
	assert.Empty(t, m2["k2"])
	assert.NotEmpty(t, m2["k1"])

	keys, err := c.HKeys("testhash")
	assert.NoError(t, err)
	assert.Len(t, keys, 2)

	err = c.HDel("testhash", "k1", "k3")
	assert.NoError(t, err)
}

func TestPubSub(t *testing.T) {
	Debug = true
	cache1, err := NewCache(Config{
		URL: os.Getenv("REDIS_MPC_URL"),
		Ctx: context.Background(),
	})
	assert.NoError(t, err)

	testType := net.MessageType("test")

	rcvd := false
	cache2, err := NewCache(Config{
		URL: os.Getenv("REDIS_MPC_URL"),
		Ctx: context.Background(),
		Handlers: net.Handlers{
			testType: func(msg net.Message) (rMsg net.Message) {
				g.Debug("received %s", msg.Type)
				rcvd = true
				return
			},
		},
	})

	cache1.Publish(cache2.PubSub.Name, net.NewMessage(testType, g.M()))
	time.Sleep(150 * time.Millisecond)
	assert.True(t, rcvd)

	// test locks
	mux1 := cache1.NewMutex(1)
	err = mux1.Lock()
	assert.NoError(t, err)
	done, err := mux1.Unlock()
	assert.NoError(t, err)
	assert.True(t, done)

	locked := false
	go func() {
		mux1.Lock()
		locked = true
		defer mux1.Unlock()
		time.Sleep(2 * time.Second)
		locked = false
	}()

	time.Sleep(300 * time.Millisecond)

	mux2 := cache2.NewMutex(1)
	assert.True(t, locked)
	mux2.Lock()
	assert.False(t, locked)
	mux2.Unlock()
}
