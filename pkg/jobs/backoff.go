package jobs

import "time"

func newBackoffTimeout(startingTimeout time.Duration, maxTimeout time.Duration, increaseFactor int) *backoff {
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
	b.current = b.current * time.Duration(b.factor)
	if b.current > b.max {
		b.current = b.max // use the max timeout
	}
}

func (b *backoff) reset() {
	b.current = b.base
}

func (b *backoff) sleep() {
	time.Sleep(b.current)
}

func (b *backoff) getCurrentTimeout() time.Duration {
	return b.current
}
