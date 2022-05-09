package util

import (
	"errors"
	"time"
)

type (
	Ticker interface {
		Chan() <-chan time.Time
		Stop()
	}

	// FakeTicker for test
	FakeTicker interface {
		Ticker
		Done()
		Tick()
		Wait(d time.Duration) error
	}

	fakeTicker struct {
		c    chan time.Time
		done chan struct{}
	}

	realTicker struct {
		*time.Ticker
	}
)

func NewTicker(d time.Duration) Ticker {
	return &realTicker{
		Ticker: time.NewTicker(d),
	}
}

// Chan implement Ticker
func (r *realTicker) Chan() <-chan time.Time {
	return r.C
}

func NewFakeTicker() FakeTicker {
	return &fakeTicker{
		c:    make(chan time.Time, 1),
		done: make(chan struct{}, 1),
	}
}

func (f *fakeTicker) Chan() <-chan time.Time {
	return f.c
}

func (f *fakeTicker) Stop() {
	close(f.c)
}

func (f *fakeTicker) Done() {
	f.done <- struct{}{}
}

func (f *fakeTicker) Tick() {
	f.c <- time.Now()
}

func (f *fakeTicker) Wait(d time.Duration) error {
	select {
	case <-time.After(d):
		return errors.New("timeout")
	case <-f.done:
		return nil
	}
}
