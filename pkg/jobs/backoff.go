package jobs

import (
	"sync"
	"time"
)

var backoffMutex sync.Mutex = sync.Mutex{} // just use a global for this one

func newBackoffTimeout(startingTimeout time.Duration, maxTimeout time.Duration, increaseFactor int) *backoff {
	backoffMutex.Lock()
	defer backoffMutex.Unlock()
	return &backoff{
		base:    startingTimeout,
		max:     maxTimeout,
		current: startingTimeout,
		factor:  increaseFactor,
	}
}

type backoff struct {
	base    time.Duration
	max     time.Duration
	current time.Duration
	factor  int
}

func (b *backoff) increaseTimeout() {
	backoffMutex.Lock()
	defer backoffMutex.Unlock()
	b.current = b.current * time.Duration(b.factor)
	if b.current > b.max {
		b.current = b.max // use the max timeout
	}
}

func (b *backoff) reset() {
	backoffMutex.Lock()
	defer backoffMutex.Unlock()
	b.current = b.base
}

func (b *backoff) sleep() {
	time.Sleep(b.current)
}

func (b *backoff) getCurrentTimeout() time.Duration {
	backoffMutex.Lock()
	defer backoffMutex.Unlock()
	return b.current
}
