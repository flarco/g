package sizedwaitgroup

import (
	"context"
	"log"
	"math"
	"os"
	"sync"
	"sync/atomic"
)

// SizedWaitGroup has the same role and close to the
// same API as the Golang sync.WaitGroup but adds a limit of
// the amount of goroutines started concurrently.
type SizedWaitGroup struct {
	Size int

	queueSize int32
	current   chan struct{}
	wg        *sync.WaitGroup
}

// New creates a SizedWaitGroup.
// The limit parameter is the maximum amount of
// goroutines which can be started concurrently.
func New(limit int) SizedWaitGroup {
	size := math.MaxInt32 // 2^32 - 1
	if limit > 0 {
		size = limit
	}
	return SizedWaitGroup{
		Size: size,

		current: make(chan struct{}, size),
		wg:      &sync.WaitGroup{},
	}
}

// Add increments the internal WaitGroup counter.
// It can be blocking if the limit of spawned goroutines
// has been reached. It will stop blocking when Done is
// been called.
//
// See sync.WaitGroup documentation for more information.
func (s *SizedWaitGroup) Add() {
	s.AddWithContext(context.Background())
}

// AddWithContext increments the internal WaitGroup counter.
// It can be blocking if the limit of spawned goroutines
// has been reached. It will stop blocking when Done is
// been called, or when the context is canceled. Returns nil on
// success or an error if the context is canceled before the lock
// is acquired.
//
// See sync.WaitGroup documentation for more information.
func (s *SizedWaitGroup) AddWithContext(ctx context.Context) error {
	if s.queueSize >= int32(s.Size) {
		if os.Getenv("DEBUG") == "TRACE" {
			log.Printf("SizedWaitGroup: %d >= %d", s.queueSize, s.Size)
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.current <- struct{}{}:
		break
	}
	atomic.AddInt32(&s.queueSize, 1)
	s.wg.Add(1)
	return nil
}

// Done decrements the SizedWaitGroup counter.
// See sync.WaitGroup documentation for more information.
func (s *SizedWaitGroup) Done() {
	if s.queueSize <= 0 {
		if os.Getenv("DEBUG") == "TRACE" {
			log.Printf("SizedWaitGroup: queueSize is %d! Calling Done() freezes...\n", s.queueSize)
		}
	}
	<-s.current
	atomic.AddInt32(&s.queueSize, -1)
	s.wg.Done()
}

// Wait blocks until the SizedWaitGroup counter is zero.
// See sync.WaitGroup documentation for more information.
func (s *SizedWaitGroup) Wait() {
	s.wg.Wait()
}

// GetQueueSize returns the number of items
// currently in the waitgroup
func (s *SizedWaitGroup) GetQueueSize() int32 {
	return s.queueSize
}
