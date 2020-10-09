package cache

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	dbURL = os.Getenv("POSTGRES_URL")
)

func TestPubSub(t *testing.T) {
	c := NewCache(dbURL)
	payload := "1234567"

	received := false
	callback := func(p string) {
		assert.Equal(t, p, payload)
		received = true
	}
	err := c.Subscribe("test_chan", callback)
	assert.NoError(t, err)

	err = c.Publish("test_chan", payload)
	assert.NoError(t, err)
	time.Sleep(300 * time.Millisecond)
	assert.True(t, received)
}

func TestLock(t *testing.T) {
	var lockCnt int
	c := NewCache(dbURL)

	assert.NoError(t, c.Lock(LockType(1)))
	assert.True(t, c.LockTry(LockType(2)))
	c.db.Get(&lockCnt, "SELECT count(1) from pg_locks where locktype = 'advisory'")
	assert.Equal(t, 2, lockCnt)

	assert.True(t, c.Unlock(LockType(1)))
	assert.True(t, c.Unlock(LockType(2)))

	c.db.Get(&lockCnt, "SELECT count(1) from pg_locks where locktype = 'advisory'")
	assert.Equal(t, 0, lockCnt)

}
