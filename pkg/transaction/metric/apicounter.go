package metric

import (
	"math"
	"sync"
)

// apiCounter tracks the count, min, max, and average response time for a group of API transactions
type apiCounter struct {
	mutex sync.Mutex
	count int64
	min   int64
	max   int64
	sum   float64
}

func newAPICounter() *apiCounter {
	return &apiCounter{}
}

// Update adds a single response time.
func (a *apiCounter) Update(responseTime int64) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.merge(1, responseTime, responseTime, float64(responseTime))
}

// UpdateWithAverage adds a batch of count transactions known only by their average response time.
func (a *apiCounter) UpdateWithAverage(count int64, avg float64) {
	if count <= 0 {
		return
	}
	sample := int64(math.Round(avg))
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.merge(count, sample, sample, avg*float64(count))
}

// UpdateWithStats adds a batch of count transactions with known min, max, and average response time.
func (a *apiCounter) UpdateWithStats(count, min, max int64, avg float64) {
	if count <= 0 {
		return
	}
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.merge(count, min, max, avg*float64(count))
}

// merge folds a batch of count transactions into the running totals. Caller must hold the mutex.
func (a *apiCounter) merge(count, min, max int64, sum float64) {
	if a.count == 0 || min < a.min {
		a.min = min
	}
	if a.count == 0 || max > a.max {
		a.max = max
	}
	a.count += count
	a.sum += sum
}

// Count returns the number of transactions counted.
func (a *apiCounter) Count() int64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.count
}

// Min returns the minimum response time recorded.
func (a *apiCounter) Min() int64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.min
}

// Max returns the maximum response time recorded.
func (a *apiCounter) Max() int64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.max
}

// Mean returns the average response time recorded.
func (a *apiCounter) Mean() float64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.count == 0 {
		return 0
	}
	return a.sum / float64(a.count)
}

// Clear resets the counter.
func (a *apiCounter) Clear() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.count = 0
	a.min = 0
	a.max = 0
	a.sum = 0
}
