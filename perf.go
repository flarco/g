package g

import (
	"time"

	"github.com/spf13/cast"
)

type TickMeasure struct {
	start time.Time
	time  time.Time
	end   time.Time
}

func NewTickMeasure() *TickMeasure {
	return &TickMeasure{
		start: time.Now(),
		time:  time.Now(),
	}
}

func (tm *TickMeasure) deltaString(d int64) string {
	switch {
	case d < 1000:
		return F("%dns", d)
	case d < 1000000:
		return F("%dus", d/1000)
	case d < 1000000000:
		return F("%dms", d/1000000)
	default:
		return F("%.3fs", cast.ToFloat64(d)/1000000000)
	}
	return F("%dns", d)
}

func (tm *TickMeasure) Tick() time.Time {
	caller := getCaller(2, 1)
	now := time.Now()
	delta := now.UnixNano() - tm.time.UnixNano()
	total := now.UnixNano() - tm.start.UnixNano()

	args := []any{
		tm.deltaString(delta),
		tm.deltaString(total),
		M("caller", caller),
	}
	doLog(LogErr.Debug(), "delta = %s | whole = %s", args)

	tm.time = now
	return now
}
