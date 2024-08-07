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
	Ctx      context.Context
	Cancel   context.CancelFunc
	ErrGroup ErrorGroup
	Wg       SizedWaitGroup
	Mux      *sync.Mutex
	LockChn  chan struct{}
	MsgChan  chan map[string]any
	Map      cmap.ConcurrentMap[string, any]
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
	return Context{Ctx: ctx, Cancel: cancel, Wg: wg, Mux: &sync.Mutex{}, ErrGroup: ErrorGroup{}, LockChn: make(chan struct{}), MsgChan: make(chan map[string]any), Map: cmap.New[any]()}
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
