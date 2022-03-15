package cacheredis

import (
	"context"
	"time"

	"github.com/flarco/g"
	"github.com/flarco/g/net"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/spf13/cast"
)

// Debug global var to debug
var Debug bool

// Config is the config for redis
type Config struct {
	URL      string
	Name     string
	Handlers net.Handlers
	Ctx      context.Context
}

// Cache is the redis cache layer
type Cache struct {
	Context *g.Context
	R       *redis.Client
	Rs      *redsync.Redsync
	PubSub  *PubSub
	GMux    *redsync.Mutex
	URL     *net.URL
}

// NewCache creates and initializes the cache service
func NewCache(cfg Config) (c *Cache, err error) {
	context := g.NewContext(cfg.Ctx)

	u, err := net.NewURL(cfg.URL)
	if err != nil {
		err = g.Error(err, "invalid redis URL")
		return
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     u.U.Host,
		Password: u.Password(), // no password set
		DB:       cast.ToInt(u.GetParam("db")),
	})

	result := rdb.Ping(context.Ctx)
	if err = result.Err(); err != nil {
		err = g.Error(err, "could not connect to redis cache")
		return
	}

	// Create an instance of redsync to be used to obtain a mutual exclusion
	// lock.
	pool := goredis.NewPool(rdb)
	rs := redsync.New(pool)
	mutex := rs.NewMutex("global-mutex")

	g.Info("connected to redis (%s)", cfg.Name)

	if cfg.Name == "" {
		cfg.Name = g.RandSuffix("pubsub-", 4)
	}

	c = &Cache{
		Context: &context,
		R:       rdb,
		Rs:      rs,
		GMux:    mutex,
		URL:     u,
	}
	c.PubSub = c.Subscribe(cfg.Name, cfg.Handlers)

	return
}

// Ctx returns the cache context
func (c *Cache) Ctx() context.Context {
	return c.Context.Ctx
}

// Close closes the connection
func (c *Cache) Close() {
	c.PubSub.PubSub.Close()
	c.R.Close()
}

// NewMutex creates a mutex
func (c *Cache) NewMutex(name interface{}) *redsync.Mutex {
	return c.Rs.NewMutex(cast.ToString(name))
}

// Publish publishes a message
func (c *Cache) Publish(topic string, msg net.Message) error {
	sent := c.R.Publish(c.Ctx(), topic, msg.JSON())
	if sent.Err() != nil {
		return g.Error(sent.Err(), "could not publish msg to %s", topic)
	}
	return nil
}

// PublishWait publishes a message and wait
func (c *Cache) PublishWait(topic string, msg net.Message, timeOut ...int) (rMsg net.Message, err error) {
	return c.PubSub.PublishWait(topic, msg, timeOut...)
}

// PublishContext publishes a message with context
func (c *Cache) PublishContext(ctx context.Context, topic string, msg net.Message) error {
	sent := c.R.Publish(ctx, topic, msg.JSON())
	return sent.Err()
}

// Subscribe creates a new pub/sub
func (c *Cache) Subscribe(name string, handlers net.Handlers) *PubSub {
	ps := &PubSub{
		Name:          name,
		PubSub:        c.R.Subscribe(c.Ctx(), name),
		Handlers:      handlers,
		ReplyHandlers: map[string]net.Handler{},
		c:             c,
	}
	c.printDebug(ps.PubSub)
	go ps.Loop()
	return ps
}

func (c *Cache) printDebug(status interface{}) {
	if Debug {
		str := ""
		switch status.(type) {
		case *redis.StatusCmd:
			str = status.(*redis.StatusCmd).String()
		case *redis.IntCmd:
			str = status.(*redis.IntCmd).String()
		case *redis.StringCmd:
			str = status.(*redis.StringCmd).String()
		case *redis.StringStringMapCmd:
			str = status.(*redis.StringStringMapCmd).String()
		case *redis.StringSliceCmd:
			str = status.(*redis.StringSliceCmd).String()
		case *redis.BoolCmd:
			str = status.(*redis.BoolCmd).String()
		case *redis.PubSub:
			str = status.(*redis.PubSub).String()
		default:
			g.Warn("did not handle printDebug for status: %#v", status)
			return
		}
		g.Debug(str)
	}
}

// Set save a key/value pair into the designated cache table
func (c *Cache) Set(key string, value interface{}) (err error) {
	return c.SetContext(c.Context.Ctx, key, value)
}

// SetM save a key/value pair into the designated cache table
func (c *Cache) SetM(key string, value map[string]interface{}) (err error) {
	return c.SetContext(c.Context.Ctx, key, value)
}

// SetEx save a key/value pair into the designated cache table which expires after a specified time
func (c *Cache) SetEx(key string, value interface{}, expire int) (err error) {
	return c.SetContext(c.Context.Ctx, key, value, expire)
}

// SetExM save a key/value pair into the designated cache table which expires after a specified time
func (c *Cache) SetExM(key string, value map[string]interface{}, expire int) (err error) {
	return c.SetContext(c.Context.Ctx, key, value, expire)
}

// SetContext save a key/value pair into the designated cache table with context
func (c *Cache) SetContext(ctx context.Context, key string, value interface{}, expire ...int) (err error) {
	var expireDuration time.Duration
	if len(expire) > 0 {
		expireDuration = time.Duration(expire[0]) * time.Second
	}

	status := c.R.Set(ctx, key, g.Marshal(value), expireDuration)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not put value for %s", key)
		return
	}

	return
}

// Has checks if a key exists in the designated cache table
func (c *Cache) Has(key string) (found bool, err error) {
	found, err = c.HasContext(c.Context.Ctx, key)
	if err != nil {
		err = g.Error(err, "could not check value for %s", key)
		return
	}
	return
}

// HasContext checks if a key exists in the designated cache table with context
func (c *Cache) HasContext(ctx context.Context, keys ...string) (found bool, err error) {
	status := c.R.Exists(ctx, keys...)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not check value")
	} else if res, _ := status.Result(); cast.ToInt(res) == len(keys) {
		found = true
	}
	return
}

// Get get a key/value pair from the designated cache table
func (c *Cache) Get(key string, valuePtr interface{}) (err error) {
	val, err := c.GetContext(c.Context.Ctx, key)
	if err != nil {
		err = g.Error(err, "could not get value for %s", key)
	}
	err = g.Unmarshal(val, valuePtr)
	if err != nil {
		err = g.Error(err, "could not unmarshal map value for %s", key)
	}
	return
}

// GetM get a key/value pair from the designated cache table
func (c *Cache) GetM(key string) (value map[string]interface{}, err error) {
	val, err := c.GetContext(c.Context.Ctx, key)
	if err != nil {
		err = g.Error(err, "could not get map value for %s", key)
	}
	err = g.Unmarshal(val, &value)
	if err != nil {
		err = g.Error(err, "could not unmarshal map value for %s", key)
	}
	return
}

// GetContext get a key/value pair from the designated cache with context
func (c *Cache) GetContext(ctx context.Context, key string) (val string, err error) {
	status := c.R.Get(ctx, key)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not get value for %s", key)
	} else {
		val = status.Val()
	}
	return
}

// Del deletes keys
func (c *Cache) Del(keys ...string) (err error) {
	err = c.DelContext(c.Context.Ctx, keys...)
	if err != nil {
		err = g.Error(err, "could not del keys: %#v", keys)
	}
	return
}

// DelContext deletes keys
func (c *Cache) DelContext(ctx context.Context, keys ...string) (err error) {
	status := c.R.Del(ctx, keys...)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not del keys: %#v", keys)
	}
	return
}

// Pop get a key/value pair from the designated cache table after deleting it
func (c *Cache) Pop(key string, valuePtr interface{}) (err error) {
	val, err := c.GetContext(c.Context.Ctx, key)
	if err != nil {
		err = g.Error(err, "could not get value for %s", key)
	}
	c.DelContext(c.Context.Ctx, key)
	err = g.Unmarshal(val, valuePtr)
	if err != nil {
		err = g.Error(err, "could not unmarshal map value for %s", key)
	}

	return
}

// HSetAll sets into a hash
func (c *Cache) HSetAll(hash string, m map[string]interface{}) (err error) {
	for k, v := range m {
		m[k] = g.Marshal(v)
	}

	status := c.R.HSet(c.Context.Ctx, hash, m)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not put hash map value for %s", hash)
		return
	}
	return
}

// HGetAll gets all from a hash
func (c *Cache) HGetAll(hash string) (m map[string]string, err error) {
	m = map[string]string{}

	status := c.R.HGetAll(c.Context.Ctx, hash)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not get hash map value for %s", hash)
		return
	}

	ms, _ := status.Result()
	for k, v := range ms {
		val := ""
		g.Unmarshal(v, &val)
		m[k] = val
	}

	return
}

// HSet sets into a hash
func (c *Cache) HSet(hash, key string, value interface{}) (err error) {
	status := c.R.HSet(c.Context.Ctx, hash, key, g.Marshal(value))
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not put value for %s", key)
		return
	}
	return
}

// HGet gets from a hash
func (c *Cache) HGet(hash, key string, valuePtr interface{}) (err error) {
	status := c.R.HGet(c.Context.Ctx, hash, key)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not get value for %s", key)
		return
	}

	err = g.Unmarshal(status.Val(), valuePtr)
	if err != nil {
		err = g.Error(err, "could not unmarshal value for hash %s %s", hash, key)
	}
	return
}

// HPop pops a key from a hash
func (c *Cache) HPop(hash, key string, valuePtr interface{}) (err error) {
	err = c.HGet(hash, key, valuePtr)
	if err != nil {
		err = g.Error(err, "could not get value for hash %s %s", hash, key)
	}

	c.HDel(hash, key)
	return
}

// HDel deletes keys of as hash
func (c *Cache) HDel(hash string, keys ...string) (err error) {
	status := c.R.HDel(c.Context.Ctx, hash, keys...)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not del keys for hash: %#v", hash, keys)
	}
	return
}

// HKeys returns all the keys for a hash
func (c *Cache) HKeys(hash string) (keys []string, err error) {
	status := c.R.HKeys(c.Context.Ctx, hash)
	c.printDebug(status)
	if err = status.Err(); err != nil {
		err = g.Error(err, "could not del keys for hash: %#v", hash, keys)
		return
	}
	keys, _ = status.Result()
	return
}

// HHas checks if a key exists in the hash
func (c *Cache) HHas(hash, key string) (found bool) {
	status := c.R.HExists(c.Context.Ctx, hash, key)
	c.printDebug(status)
	found, _ = status.Result()
	return
}
