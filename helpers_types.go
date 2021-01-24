package g

import (
	"context"
	"os"
	"strings"
	"sync"

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
}

// SizedWaitGroup with separate wait groups for read & write
type SizedWaitGroup struct {
	Read  sizedwaitgroup.SizedWaitGroup
	Write sizedwaitgroup.SizedWaitGroup
	Limit int
}

const defaultConcurencyLimit = 10

// NewContext creates a new context
func NewContext(parentCtx context.Context, concurencyLimits ...int) Context {
	concurencyLimit := defaultConcurencyLimit
	if len(concurencyLimits) > 0 {
		concurencyLimit = concurencyLimits[0]
	} else if os.Getenv("_CONCURENCY_LIMIT") != "" {
		concurencyLimit = cast.ToInt(os.Getenv("_CONCURENCY_LIMIT"))
	}
	ctx, cancel := context.WithCancel(parentCtx)
	wg := SizedWaitGroup{
		Limit: concurencyLimit,
		Read:  sizedwaitgroup.New(concurencyLimit),
		Write: sizedwaitgroup.New(concurencyLimit),
	}
	return Context{Ctx: ctx, Cancel: cancel, Wg: wg, Mux: &sync.Mutex{}, ErrGroup: ErrorGroup{}}
}

// SetConcurencyLimit sets the concurency limit
func (c *Context) SetConcurencyLimit(concurencyLimit int) {
	c.Wg = SizedWaitGroup{
		Limit: concurencyLimit,
		Read:  sizedwaitgroup.New(concurencyLimit),
		Write: sizedwaitgroup.New(concurencyLimit),
	}
}

// CaptureErr if err != nil, captures the error from concurent function
func (c *Context) CaptureErr(E error, args ...interface{}) bool {
	if E != nil {
		if !strings.Contains(E.Error(), " --- at ") && IsDebugLow() {
			msg := ArgsErrMsg(args...)
			E = NewError(3, E, msg)
		}
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
