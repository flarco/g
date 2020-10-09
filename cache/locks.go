package cache

import (
	g "github.com/flarco/gutil"
)

// LockType is the type of Lock
type LockType int

const (
	LockTypeMin LockType = iota

	LockTypeMax
)

// Lock waits obtain an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) Lock(lockID LockType) (err error) {
	_, err = c.db.ExecContext(c.Context.Ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		err = g.Error(err, "could not obtain advisory_lock")
	}
	return
}

// LockTry do not wait to obtain an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) LockTry(lockID LockType) (success bool) {
	err := c.db.Get(&success, "SELECT pg_try_advisory_lock($1)", lockID)
	g.LogError(err)
	return
}

// Unlock releases an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) Unlock(lockID LockType) (success bool) {
	err := c.db.Get(&success, "SELECT pg_advisory_unlock($1)", lockID)
	g.LogError(err)
	return
}
