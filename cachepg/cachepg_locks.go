package cachepg

import (
	"context"
	"database/sql"

	g "github.com/flarco/gutil"
)

// LockType is the type of Lock
type LockType int

// Lock waits obtain an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) Lock(lockID LockType) (tx *sql.Tx, err error) {
	return c.LockContext(c.Context.Ctx, lockID)
}

// LockContext waits obtain an advisory lock on PG (distributed locking) with context
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) LockContext(ctx context.Context, lockID LockType) (tx *sql.Tx, err error) {
	tx, err = c.db.Begin()
	if err != nil {
		err = g.Error(err, "could not begin transaction")
		tx = nil
		return
	}
	_, err = tx.Stmt(c.lockStmt).ExecContext(ctx, lockID)
	if err != nil {
		err = g.Error(err, "could not obtain advisory_lock")
	}
	return
}

// LockTry do not wait to obtain an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) LockTry(lockID LockType) (tx *sql.Tx, success bool) {
	tx, err := c.db.Begin()
	if err != nil {
		tx = nil
		return
	}
	row := tx.Stmt(c.lockTryStmt).QueryRow(lockID)
	err = row.Scan(&success)
	g.LogError(err, "could not lock")
	return
}

// Unlock releases an advisory lock on PG (distributed locking)
// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS
func (c *Cache) Unlock(tx *sql.Tx, lockID LockType) (success bool) {
	if c.closed || tx == nil {
		return
	}
	row := tx.Stmt(c.unlockStmt).QueryRow(lockID)
	err := row.Scan(&success)
	g.LogError(err, "could not unlock")
	tx.Commit()
	return
}
