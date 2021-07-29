package metric

import (
	"sync/atomic"

	metrics "github.com/rcrowley/go-metrics"
)

// metricCounter hold an int64 count and usage values that can be incremented and decremented.
type metricCounter interface {
	Clear()
	Count() int64
	DecCount(int64)
	IncCount(int64)
	Volume() int64
	DecVolume(int64)
	IncVolume(int64)
	Snapshot() metricCounter
}

// getOrRegisterCounter - returns an existing metricCounter or constructs and registers a new standardMetricCounter.
func getOrRegisterCounter(name string, r metrics.Registry) metricCounter {
	if nil == r {
		r = metrics.DefaultRegistry
	}
	return r.GetOrRegister(name, newMetricCounter).(metricCounter)
}

// newMetricCounter - constructs a new standardMetricCounter.
func newMetricCounter() metricCounter {
	return &standardMetricCounter{
		&metricCounterSnapshot{}, 0, 0}
}

// newRegisteredCounter - constructs and registers a new standardMetricCounter.
func newRegisteredCounter(name string, r metrics.Registry) metricCounter {
	c := newMetricCounter()
	if nil == r {
		r = metrics.DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// metricCounterSnapshot - is a read-only copy of another metricCounter.
type metricCounterSnapshot struct {
	count  int64
	volume int64
}

// Clear panics.
func (metricCounterSnapshot) Clear() {
	panic("Clear called on a metricCounterSnapshot")
}

// Count returns the count at the time the snapshot was taken.
func (s *metricCounterSnapshot) Count() int64 { return int64(s.count) }

// Dec panics.
func (metricCounterSnapshot) DecCount(int64) {
	panic("Dec called on a metricCounterSnapshot")
}

// Inc panics.
func (metricCounterSnapshot) IncCount(int64) {
	panic("Inc called on a metricCounterSnapshot")
}

// Count returns the count at the time the snapshot was taken.
func (s *metricCounterSnapshot) Volume() int64 { return int64(s.volume) }

// Dec panics.
func (metricCounterSnapshot) DecVolume(int64) {
	panic("DecVolume called on a metricCounterSnapshot")
}

// Inc panics.
func (metricCounterSnapshot) IncVolume(int64) {
	panic("IncVolume called on a metricCounterSnapshot")
}

// Snapshot returns the snapshot.
func (s *metricCounterSnapshot) Snapshot() metricCounter { return s }

// standardMetricCounter - is the standard implementation of a metricCounter and uses the
// sync/atomic package to manage a single int64 value.
type standardMetricCounter struct {
	snapshot *metricCounterSnapshot
	count    int64
	volume   int64
}

// Clear - sets the counter to zero.
func (c *standardMetricCounter) Clear() {
	atomic.StoreInt64(&c.count, 0)
	atomic.StoreInt64(&c.volume, 0)
}

// Count - returns the current count.
func (c *standardMetricCounter) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

// DecCount - decrements the counter by the given amount.
func (c *standardMetricCounter) DecCount(i int64) {
	atomic.AddInt64(&c.count, -i)
}

// IncCount - increments the counter by the given amount.
func (c *standardMetricCounter) IncCount(i int64) {
	atomic.AddInt64(&c.count, i)
}

// Volume - returns the current volume.
func (c *standardMetricCounter) Volume() int64 {
	return atomic.LoadInt64(&c.volume)
}

// DecVolume - decrements the volume by the given amount.
func (c *standardMetricCounter) DecVolume(i int64) {
	atomic.AddInt64(&c.volume, -i)
}

// IncVolume - increments the volume by the given amount.
func (c *standardMetricCounter) IncVolume(i int64) {
	atomic.AddInt64(&c.volume, i)
}

// Snapshot returns a read-only copy of the meter.
func (c *standardMetricCounter) Snapshot() metricCounter {
	copiedSnapshot := metricCounterSnapshot{
		count:  atomic.LoadInt64(&c.snapshot.count),
		volume: atomic.LoadInt64(&c.snapshot.volume),
	}
	return &copiedSnapshot
}
