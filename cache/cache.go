package cache

import (
	"context"
	"database/sql"
	"sync"
	"time"

	g "github.com/flarco/gutil"
	"github.com/jmoiron/sqlx"
	cmap "github.com/orcaman/concurrent-map"
)

// the key to user for single value maps
const valKey = "v"

// TableName is the table name to use for the cache table
var TableName = "__cache__"

// TableDDL is the table DDL if manually creating
var TableDDL = g.R(`
	CREATE TABLE IF NOT EXISTS {table} (
		"key" text NOT NULL,
		value jsonb NOT NULL DEFAULT '{}'::jsonb,
		expire_dt timestamp NULL,
		updated_dt timestamp NULL,
		CONSTRAINT caches_pkey PRIMARY KEY (key)
	);
	CREATE INDEX IF NOT EXISTS idx_cache_expire_dt ON {table} USING btree (expire_dt);
	`, "table", TableName,
)

// Table represents the cache table, for caching
type Table struct {
	Key       string     `json:"key" gorm:"primaryKey"`
	Value     string     `json:"value" gorm:"type:jsonb not null default '{}'::jsonb"`
	ExpireDt  *time.Time `json:"expire_dt"  gorm:"index:idx_cache_expire_dt"`
	UpdatedDt time.Time  `json:"updated_dt" gorm:"autoUpdateTime"`
}

// TableName overrides the table name used in gorm
func (Table) TableName() string {
	return TableName
}

// Cache is a Postgres Cache Backend
type Cache struct {
	Context    g.Context
	mux        sync.Mutex
	db         *sqlx.DB
	listeners  cmap.ConcurrentMap
	defChannel string // Default listener channel
	dbURL      string
	setStmt    *sql.Stmt
	getStmt    *sql.Stmt
}

// NewCache creates a new cache instance
func NewCache(dbURL string) (c *Cache, err error) {
	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		err = g.Error(err, "Could not initialize cache database connection")
		return
	}

	c = &Cache{
		Context:   g.NewContext(context.Background()),
		db:        db,
		dbURL:     dbURL,
		listeners: cmap.New(),
	}

	err = c.createTable()
	if err != nil {
		return c, g.Error(err, "could not create cache table")
	}

	// create default listener
	c.subscribeDefault()

	return c, nil
}

// Db returns the db connection
func (c *Cache) Db() *sqlx.DB {
	return c.db
}

// Close closes the cache connection
func (c *Cache) Close() {
	defer c.Context.Cancel()
	for _, listenerI := range c.listeners.Items() {
		listener, ok := listenerI.(*Listener)
		if ok {
			listener.Close()
		}
	}
	c.db.Close()
}

func (c *Cache) createTable() (err error) {
	_, err = c.db.Exec(TableDDL)
	if err != nil {
		err = g.Error(err, "could not create cache table")
	}
	return
}

func (c *Cache) dropTable() (err error) {
	_, err = c.db.Exec(g.F("drop table if exists %s", TableName))
	if err != nil {
		err = g.Error(err, "could not create cache table")
	}
	return
}

// DeleteExpired delets all expired records from cache table
func (c *Cache) DeleteExpired() (err error) {
	_, err = c.db.Exec(g.F("delete from %s where expire_dt < now()", TableName))
	if err != nil {
		err = g.Error(err, "could not delete expired rows")
	}
	return
}

// Set save a key/value pair into the designated cache table
func (c *Cache) Set(key string, value interface{}) (err error) {
	return c.SetContext(c.Context.Ctx, key, g.M(valKey, value))
}

// SetM save a key/value pair into the designated cache table
func (c *Cache) SetM(key string, value map[string]interface{}) (err error) {
	return c.SetContext(c.Context.Ctx, key, value)
}

// SetEx save a key/value pair into the designated cache table which expires after a specified time
func (c *Cache) SetEx(key string, value interface{}, expire int) (err error) {
	return c.SetContext(c.Context.Ctx, key, g.M(valKey, value), expire)
}

// SetExM save a key/value pair into the designated cache table which expires after a specified time
func (c *Cache) SetExM(key string, value map[string]interface{}, expire int) (err error) {
	return c.SetContext(c.Context.Ctx, key, value, expire)
}

// SetContext save a key/value pair into the designated cache table with context
func (c *Cache) SetContext(ctx context.Context, key string, value map[string]interface{}, expire ...int) (err error) {
	var expireDt *time.Time
	if len(expire) > 0 {
		val := time.Now().Add(time.Duration(expire[0]) * time.Second)
		expireDt = &val
	}

	if c.setStmt == nil {
		sql := g.R(
			`insert into {table} ("key", "value", "expire_dt")
			values ($1, $2, $3)
			on conflict ("key") do update set "value" = $2, "expire_dt" = $3`,
			"table", TableName,
		)

		c.setStmt, err = c.db.Prepare(sql)
		if err != nil {
			err = g.Error(err, "could not prepare statement to put value for %s", key)
			return
		}
	}

	valueStr := g.MarshalMap(value)
	_, err = c.setStmt.ExecContext(ctx, key, valueStr, expireDt)
	if err != nil {
		err = g.Error(err, "could not put value for %s", key)
		return
	}

	if expireDt != nil {
		time.AfterFunc(
			time.Until(*expireDt),
			func() { c.Pop(key) },
		)
	}

	return
}

// Get get a key/value pair from the designated cache table
func (c *Cache) Get(key string) (value interface{}, err error) {
	valM, err := c.GetContext(c.Context.Ctx, key)
	if err != nil {
		err = g.Error(err, "could not get value for %s", key)
		return
	}
	value = valM[valKey]
	return
}

// GetM get a key/value pair from the designated cache table
func (c *Cache) GetM(key string) (value map[string]interface{}, err error) {
	value, err = c.GetContext(c.Context.Ctx, key)
	if err != nil {
		err = g.Error(err, "could not get map value for %s", key)
	}
	return
}

// GetLikeKeys gets keys from the designated cache table with a key LIKE filter
func (c *Cache) GetLikeKeys(pattern string) (values []string, err error) {
	values, err = c.GetLikeKeysContext(c.Context.Ctx, pattern)
	if err != nil {
		err = g.Error(err, "could not get keys like %s", pattern)
	}
	return
}

// GetLikeValues get a key/value pair from the designated cache table with a key LIKE filter
func (c *Cache) GetLikeValues(pattern string) (values []interface{}, err error) {
	valArrM, err := c.GetLikeValuesContext(c.Context.Ctx, pattern)
	if err != nil {
		err = g.Error(err, "could not get values with keys like %s", pattern)
		return
	}
	values = make([]interface{}, len(valArrM))
	for i, m := range valArrM {
		values[i] = m[valKey]
	}
	return
}

// GetLikeValuesM get a key/value pair from the designated cache table with a key LIKE filter
func (c *Cache) GetLikeValuesM(pattern string) (values []map[string]interface{}, err error) {
	values, err = c.GetLikeValuesContext(c.Context.Ctx, pattern)
	if err != nil {
		err = g.Error(err, "could not get map values with keys like %s", pattern)
	}
	return
}

// Pop get a key/value pair from the designated cache table after deleting it
func (c *Cache) Pop(key string) (value interface{}, err error) {
	sql := g.R(
		"delete from {table} where key = $1 returning value",
		"table", TableName,
	)

	valM, err := c.GetContextSQL(c.Context.Ctx, sql, key)
	if err != nil {
		err = g.Error(err, "could not get value for %s", key)
		return
	}
	value = valM[valKey]

	return
}

// PopM get a key/value pair from the designated cache table after deleting it
func (c *Cache) PopM(key string) (value map[string]interface{}, err error) {
	sql := g.R(
		"delete from {table} where key = $1 returning value",
		"table", TableName,
	)
	value, err = c.GetContextSQL(c.Context.Ctx, sql, key)
	if err != nil {
		err = g.Error(err, "could not get map value for %s", key)
	}
	return
}

// PopLike get key/value pairs from the designated cache table after deleting them
func (c *Cache) PopLike(pattern string) (values []interface{}, err error) {
	sql := g.R(
		"delete from {table} where key LIKE $1 returning value",
		"table", TableName,
	)

	valArrM, err := c.SelectContextSQL(c.Context.Ctx, sql, pattern)
	if err != nil {
		err = g.Error(err, "could not pop values like %s", pattern)
		return
	}

	values = make([]interface{}, len(valArrM))
	for i, m := range valArrM {
		values[i] = m[valKey]
	}
	return
}

// PopLikeM get key/value pairs from the designated cache table after deleting them
func (c *Cache) PopLikeM(pattern string) (values []map[string]interface{}, err error) {
	sql := g.R(
		"delete from {table} where key LIKE $1 returning value",
		"table", TableName,
	)

	values, err = c.SelectContextSQL(c.Context.Ctx, sql, pattern)
	if err != nil {
		err = g.Error(err, "could not pop map values like %s", pattern)
	}
	return
}

// GetContext get a key/value pair from the designated cache table with context
func (c *Cache) GetContext(ctx context.Context, key string) (value map[string]interface{}, err error) {
	sql := g.R(
		"SELECT value from {table} where key = $1",
		"table", TableName,
	)
	return c.GetContextSQL(ctx, sql, key)
}

// GetLikeKeysContext get keys from the designated cache table with a key LIKE filter with context
func (c *Cache) GetLikeKeysContext(ctx context.Context, pattern string) (values []string, err error) {
	sql := g.R(
		"SELECT key from {table} where key LIKE $1",
		"table", TableName,
	)

	err = c.db.SelectContext(ctx, &values, sql, pattern)
	if err != nil {
		err = g.Error(err, "could not get key for %s", pattern)
	}

	return
}

// GetLikeValuesContext get a key/value pair from the designated cache table with a key LIKE filter with context
func (c *Cache) GetLikeValuesContext(ctx context.Context, pattern string) (values []map[string]interface{}, err error) {
	sql := g.R(
		"SELECT value from {table} where key LIKE $1",
		"table", TableName,
	)
	return c.SelectContextSQL(ctx, sql, pattern)
}

// GetContextSQL save a key/value pair from the designated cache table with context
func (c *Cache) GetContextSQL(ctx context.Context, sql, key string) (value map[string]interface{}, err error) {
	var valueStr string
	err = c.db.GetContext(ctx, &valueStr, sql, key)
	if err != nil {
		err = g.Error(err, "could not get value for %s", key)
		return
	}

	err = g.Unmarshal(valueStr, &value)
	if err != nil {
		err = g.Error(err, "could not parse value for %s", key)
	}

	return
}

// SelectContextSQL save a key/value pair from the designated cache table with context
func (c *Cache) SelectContextSQL(ctx context.Context, sql, pattern string) (values []map[string]interface{}, err error) {
	var valueArr []string
	err = c.db.SelectContext(ctx, &valueArr, sql, pattern)
	if err != nil {
		err = g.Error(err, "could not get values for %s", pattern)
		return
	}

	values = make([]map[string]interface{}, len(valueArr))
	for i, valueStr := range valueArr {
		val := g.M()
		err = g.Unmarshal(valueStr, &val)
		if err != nil {
			err = g.Error(err, "could not parse value for %s", pattern)
			break
		}
		values[i] = val
	}

	return
}
