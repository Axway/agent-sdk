package metric

import "sync/atomic"

// counter is a thread-safe counter backed by an atomic int64
type counter struct {
	count atomic.Int64
}

// newCounter creates a new counter
func newCounter() *counter {
	return &counter{}
}

// Clear resets the counter to zero
func (c *counter) Clear() {
	c.count.Store(0)
}

// Count returns the current count
func (c *counter) Count() int64 {
	return c.count.Load()
}

// Dec decrements the counter by the given amount
func (c *counter) Dec(i int64) {
	c.count.Add(-i)
}

// Inc increments the counter by the given amount
func (c *counter) Inc(i int64) {
	c.count.Add(i)
}
