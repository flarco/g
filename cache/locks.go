package cache

import (
	"context"

	g "github.com/flarco/gutil"
)

// LockType is the type of Lock
type LockType int

// Lock waits obtain an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) Lock(lockID LockType) (err error) {
	return c.LockContext(c.Context.Ctx, lockID)
}

// LockContext waits obtain an advisory lock on PG (distributed locking) with context
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) LockContext(ctx context.Context, lockID LockType) (err error) {
	_, err = c.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
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
