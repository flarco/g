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

// TableName is the table name to use for the cache table
var TableName = "__cache__"

// TableDDL is the table DDL if manually creating
var TableDDL = g.R(`
	CREATE TABLE {table} (
		"key" text NOT NULL,
		value jsonb NOT NULL DEFAULT '{}'::jsonb,
		expire_dt timestamp NULL,
		updated_dt timestamp NULL,
		CONSTRAINT caches_pkey PRIMARY KEY (key)
	);
	CREATE INDEX idx_cache_expire_dt ON {table} USING btree (expire_dt);
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
	Context   g.Context
	mux       sync.Mutex
	db        *sqlx.DB
	listeners cmap.ConcurrentMap
	dbURL     string
	setStmt   *sql.Stmt
	getStmt   *sql.Stmt
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
	return c, nil
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

// Set save a key/value pair into the designated cache table
func (c *Cache) Set(key string, value map[string]interface{}) (err error) {
	return c.SetContext(c.Context.Ctx, key, value)
}

// SetEx save a key/value pair into the designated cache table which expires after a specified time
func (c *Cache) SetEx(key string, value map[string]interface{}, expire int) (err error) {
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
func (c *Cache) Get(key string) (value map[string]interface{}, err error) {
	return c.GetContext(c.Context.Ctx, key)
}

// GetContext save a key/value pair from the designated cache table with context
func (c *Cache) GetContext(ctx context.Context, key string) (value map[string]interface{}, err error) {
	sql := g.R(
		"SELECT value from {table} where key = $1",
		"table", TableName,
	)
	return c.GetContextSQL(ctx, sql, key)
}

// Pop get a key/value pair from the designated cache table after deleting it
func (c *Cache) Pop(key string) (value map[string]interface{}, err error) {
	sql := g.R(
		"delete from {table} where key = $1 returning value",
		"table", TableName,
	)
	return c.GetContextSQL(c.Context.Ctx, sql, key)
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
