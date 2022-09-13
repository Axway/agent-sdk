package jobs

import (
	"sync"
	"time"
)

func newBackoffTimeout(startingTimeout time.Duration, maxTimeout time.Duration, increaseFactor int) *backoff {
	return &backoff{
		base:         startingTimeout,
		max:          maxTimeout,
		current:      startingTimeout,
		factor:       increaseFactor,
		backoffMutex: &sync.Mutex{},
	}
}

type backoff struct {
	base         time.Duration
	max          time.Duration
	current      time.Duration
	factor       int
	backoffMutex *sync.Mutex
}

func (b *backoff) increaseTimeout() {
	b.backoffMutex.Lock()
	defer b.backoffMutex.Unlock()
	b.current = b.current * time.Duration(b.factor)
	if b.current > b.max {
		b.current = b.base // reset to base timeout
	}
}

func (b *backoff) reset() {
	b.backoffMutex.Lock()
	defer b.backoffMutex.Unlock()
	b.current = b.base
}

func (b *backoff) sleep() {
	time.Sleep(b.current)
}

func (b *backoff) getCurrentTimeout() time.Duration {
	b.backoffMutex.Lock()
	defer b.backoffMutex.Unlock()
	return b.current
}
