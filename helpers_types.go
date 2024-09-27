package g

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/flarco/g/sizedwaitgroup"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/spf13/cast"
)

// Context is to manage context
type Context struct {
	Ctx      context.Context     `json:"-"`
	Cancel   context.CancelFunc  `json:"-"`
	ErrGroup ErrorGroup          `json:"-"`
	Wg       SizedWaitGroup      `json:"-"`
	Mux      *sync.Mutex         `json:"-"`
	LockChn  chan struct{}       `json:"-"`
	MsgChan  chan map[string]any `json:"-"`

	Map cmap.ConcurrentMap[string, any] `json:"map"`

	with map[string]any `json:"-"` // for  single log event
}

// SizedWaitGroup with separate wait groups for read & write
type SizedWaitGroup struct {
	Read  *sizedwaitgroup.SizedWaitGroup
	Write *sizedwaitgroup.SizedWaitGroup
	Limit int
}

// NewContext creates a new context
func NewContext(parentCtx context.Context, concurrencyLimits ...int) Context {
	concurrencyLimit := runtime.NumCPU()
	if len(concurrencyLimits) > 0 {
		concurrencyLimit = concurrencyLimits[0]
	} else if os.Getenv("CONCURRENCY_LIMIT") != "" {
		concurrencyLimit = cast.ToInt(os.Getenv("CONCURRENCY_LIMIT"))
	}
	ctx, cancel := context.WithCancel(parentCtx)
	wg := SizedWaitGroup{
		Limit: concurrencyLimit,
		Read:  sizedwaitgroup.New(concurrencyLimit),
		Write: sizedwaitgroup.New(concurrencyLimit),
	}
	return Context{
		Ctx:      ctx,
		Cancel:   cancel,
		Wg:       wg,
		Mux:      &sync.Mutex{},
		ErrGroup: ErrorGroup{},
		LockChn:  make(chan struct{}),
		MsgChan:  make(chan map[string]any),
		Map:      cmap.New[any](),
	}
}

const logKeyID = "_log_keys"

// Wrapper to set log values
func (c *Context) WithLogValues(KVs ...any) *Context {
	c.SetLogValues(KVs...)
	return c
}

// SetLogValues sets log key and value pairs
func (c *Context) SetLogValues(KVs ...any) {
	logKeysMap := c.getLogKeysMap()
	for k, v := range M(KVs...) {
		c.Map.Set(k, v)
		logKeysMap[k] = nil
	}
	c.Map.Set(logKeyID, logKeysMap)
}

// GetLogValues gets Log values as a map
func (c *Context) GetLogValues() map[string]any {
	m := M()
	for k := range c.getLogKeysMap() {
		m[k], _ = c.Map.Get(k)
	}

	for k, v := range c.with {
		m[k] = v
	}
	c.with = nil // reset

	return m
}

// getLogKeysMap gets log key map
func (c *Context) getLogKeysMap() (logKeysMap map[string]any) {
	if val, ok := c.Map.Get(logKeyID); ok {
		logKeysMap = val.(map[string]any)
	} else {
		logKeysMap = map[string]any{}
	}

	return logKeysMap
}

// With provides KVs for only the next log event
func (c *Context) With(KVs ...any) *Context {
	c.with = M(KVs...)
	return c
}

func (c *Context) Trace(text string, args ...any) { Trace(text, append(args, c.GetLogValues())...) }
func (c *Context) Debug(text string, args ...any) { Debug(text, append(args, c.GetLogValues())...) }
func (c *Context) Info(text string, args ...any)  { Info(text, append(args, c.GetLogValues())...) }
func (c *Context) Warn(text string, args ...any)  { Warn(text, append(args, c.GetLogValues())...) }
func (c *Context) Error(text string, args ...any) { Err(text, append(args, c.GetLogValues())...) }
func (c *Context) LogError(E error, args ...interface{}) {
	LogError(E, append(args, c.GetLogValues())...)
}

// SetConcurrencyLimit sets the concurrency limit
func (c *Context) SetConcurrencyLimit(concurrencyLimit int) {
	c.Wg = SizedWaitGroup{
		Limit: concurrencyLimit,
		Read:  sizedwaitgroup.New(concurrencyLimit),
		Write: sizedwaitgroup.New(concurrencyLimit),
	}
}

// MemBasedLimit limit the concurrency based on mem
func (c *Context) MemBasedLimit(percentLimit int) {
	stats := GetMachineProcStats()
	if ramPct := cast.ToInt(stats.RamPct); ramPct > percentLimit {
		// loop until memory is low again
		Warn("Memory based limit applied. High RAM detected: %d%%. Consider lowering CONCURRENCY setting.", ramPct)
		for {
			time.Sleep(2 * time.Second)
			ramPct = cast.ToInt(GetMachineProcStats().RamPct)
			if ramPct < percentLimit {
				break
			}
		}
		Info("Memory based limit released. Current RAM: %d%", ramPct)
	}
}

// CaptureErr if err != nil, captures the error from concurent function
// and cancels the context
func (c *Context) CaptureErr(E error, args ...interface{}) bool {
	if E != nil {
		Trace("Context.CaptureErr => %#v", E)
		if GetLogLevel() == TraceLevel {
			LogError(E)
		}
		E = NewError(3, E, args...)
		c.Cancel() // cancel context
	}
	return c.ErrGroup.Capture(E)
}

// Err return error if any
func (c *Context) Err() error {
	if c.Ctx.Err() != nil {
		c.ErrGroup.Capture(c.Ctx.Err())
	}
	return c.ErrGroup.Err()
}

// Lock locks mutex
func (c *Context) Lock() {
	c.Mux.Lock()
}

// Unlock unlocks mutex
func (c *Context) Unlock() {
	c.Mux.Unlock()
}
