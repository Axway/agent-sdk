package jobs

import (
	"sync/atomic"
	"time"
)

func newBackoffTimeout(startingTimeout time.Duration, maxTimeout time.Duration, increaseFactor int) *backoff {
	b := &backoff{
		factor: increaseFactor,
	}
	b.base.Store(startingTimeout)
	b.max.Store(maxTimeout)
	b.current.Store(startingTimeout)
	return b
}

type backoff struct {
	base    atomic.Value // atomic.Value to store the base timeout (thread-safe)
	max     atomic.Value // atomic.Value to store the max timeout (thread-safe)
	current atomic.Value // atomic.Value to store the current timeout (thread-safe)
	factor  int          // multiplier factor for increasing the timeout
}

func (b *backoff) increaseTimeout() {
	current := b.getCurrentTimeout()
	newTimeout := current * time.Duration(b.factor)
	if newTimeout > b.getMaxTimeout() {
		newTimeout = b.getBaseTimeout() // reset to base timeout
	}
	b.current.Store(newTimeout)
}

func (b *backoff) reset() {
	b.current.Store(b.getBaseTimeout())
}

func (b *backoff) sleep() {
	time.Sleep(b.getCurrentTimeout())
}

func (b *backoff) getCurrentTimeout() time.Duration {
	return b.current.Load().(time.Duration)
}

func (b *backoff) getBaseTimeout() time.Duration {
	return b.base.Load().(time.Duration)
}

func (b *backoff) getMaxTimeout() time.Duration {
	return b.max.Load().(time.Duration)
}
