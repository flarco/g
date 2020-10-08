package cache

import (
	"context"
	"sync"

	g "github.com/flarco/gutil"
	"github.com/jmoiron/sqlx"
	cmap "github.com/orcaman/concurrent-map"
)

// Cache is a Postgres Cache Backend
type Cache struct {
	Context   g.Context
	mux       sync.Mutex
	db        *sqlx.DB
	listeners cmap.ConcurrentMap
	dbURL     string
}

// NewCache creates a new cache instance
func NewCache(dbURL string) *Cache {
	db, err := sqlx.Open("postgres", dbURL)
	g.LogFatal(err, "Could not initialize cache database connection")

	c := Cache{
		Context:   g.NewContext(context.Background()),
		db:        db,
		dbURL:     dbURL,
		listeners: cmap.New(),
	}
	return &c
}
