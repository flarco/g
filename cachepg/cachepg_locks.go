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
	_, err = c.lockStmt.ExecContext(ctx, lockID)
	if err != nil {
		err = g.Error(err, "could not obtain advisory_lock")
	}
	return
}

// LockTry do not wait to obtain an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) LockTry(lockID LockType) (success bool) {
	row := c.lockTryStmt.QueryRow(lockID)
	err := row.Scan(&success)
	g.LogError(err, "could not lock")
	return
}

// Unlock releases an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) Unlock(lockID LockType) (success bool) {
	if c.closed {
		return
	}
	row := c.unlockStmt.QueryRow(lockID)
	err := row.Scan(&success)
	g.LogError(err, "could not unlock")
	return
}
