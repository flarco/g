package g

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/flarco/g/sizedwaitgroup"
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
}

// SizedWaitGroup with separate wait groups for read & write
type SizedWaitGroup struct {
	Read  *sizedwaitgroup.SizedWaitGroup
	Write *sizedwaitgroup.SizedWaitGroup
	Limit int
}

// NewContext creates a new context
func NewContext(parentCtx context.Context, concurencyLimits ...int) Context {
	concurencyLimit := runtime.NumCPU()
	if len(concurencyLimits) > 0 {
		concurencyLimit = concurencyLimits[0]
	} else if os.Getenv("CONCURENCY_LIMIT") != "" {
		concurencyLimit = cast.ToInt(os.Getenv("CONCURENCY_LIMIT"))
	}
	ctx, cancel := context.WithCancel(parentCtx)
	wg := SizedWaitGroup{
		Limit: concurencyLimit,
		Read:  sizedwaitgroup.New(concurencyLimit),
		Write: sizedwaitgroup.New(concurencyLimit),
	}
	return Context{Ctx: ctx, Cancel: cancel, Wg: wg, Mux: &sync.Mutex{}, ErrGroup: ErrorGroup{}, LockChn: make(chan struct{})}
}

// SetConcurencyLimit sets the concurency limit
func (c *Context) SetConcurencyLimit(concurencyLimit int) {
	c.Wg = SizedWaitGroup{
		Limit: concurencyLimit,
		Read:  sizedwaitgroup.New(concurencyLimit),
		Write: sizedwaitgroup.New(concurencyLimit),
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
		E = NewError(3, E, args...)
		c.Cancel() // cancel context
	}
	return c.ErrGroup.Capture(E)
}

// Err return error if any
func (c *Context) Err() error {
	if c.Ctx.Err() != nil {
		eg := ErrorGroup{Errors: []error{c.Ctx.Err()}}
		eg.Capture(c.ErrGroup.Err())
		return eg.Err()
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
