package cache

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/cast"

	g "github.com/flarco/gutil"
	"github.com/stretchr/testify/assert"
)

var (
	dbURL = os.Getenv("POSTGRES_URL")
)

func TestPubSub(t *testing.T) {
	c, err := NewCache(dbURL)
	assert.NoError(t, err)
	payload := "1234567"

	received := false
	callback := func(p string) {
		assert.Equal(t, p, payload)
		received = true
	}
	err = c.Subscribe("test_chan", callback)
	assert.NoError(t, err)

	err = c.Publish("test_chan", payload)
	assert.NoError(t, err)
	time.Sleep(300 * time.Millisecond)
	assert.True(t, received)
}

func TestLock(t *testing.T) {
	var lockCnt int
	c, err := NewCache(dbURL)
	assert.NoError(t, err)

	assert.NoError(t, c.Lock(LockType(1)))
	assert.True(t, c.LockTry(LockType(2)))
	c.db.Get(&lockCnt, "SELECT count(1) from pg_locks where locktype = 'advisory'")
	assert.Equal(t, 2, lockCnt)

	assert.True(t, c.Unlock(LockType(1)))
	assert.True(t, c.Unlock(LockType(2)))

	c.db.Get(&lockCnt, "SELECT count(1) from pg_locks where locktype = 'advisory'")
	assert.Equal(t, 0, lockCnt)

}

func TestSetGet(t *testing.T) {
	c, err := NewCache(dbURL)
	assert.NoError(t, err)

	err = c.createTable()
	if !assert.NoError(t, err) {
		return
	}
	defer c.dropTable()

	err = c.Set("key-1", g.M("error", "a stupid one"))
	assert.NoError(t, err)

	arr := []interface{}{1, "a", true}
	m := g.M("a", arr)
	err = c.Set("key-2", g.M("nested", m, "arr", arr))
	assert.NoError(t, err)

	vals, err := c.GetLike("key-%")
	assert.Len(t, vals, 2)

	val, err := c.Get("key-1")
	assert.Equal(t, "a stupid one", val["error"])

	err = c.SetEx("key-1", g.M("error", "another stupid one"), 1)
	assert.NoError(t, err)

	val, err = c.Get("key-1")
	assert.Equal(t, "another stupid one", val["error"])

	time.Sleep(2 * time.Second)

	// should not be there since expired
	_, err = c.Get("key-1")
	assert.Error(t, err)

	val, err = c.Pop("key-2")
	assert.NoError(t, err)

	arr2 := cast.ToStringMap(val["nested"])["a"]
	assert.Equal(t, cast.ToString(arr), cast.ToString(arr2))
	assert.Equal(t, cast.ToString(arr), cast.ToString(val["arr"]))

	// should not be there since popped
	_, err = c.Get("key-2")
	assert.Error(t, err)
}
