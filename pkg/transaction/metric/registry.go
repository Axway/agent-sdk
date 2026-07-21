package metric

import (
	"fmt"
	"sync"
)

// The metric registry which can store counters or grouped metrics
type registry interface {
	Each(func(string, interface{}))
	Get(string) interface{}
	Register(string, interface{}) error
	Deregister(string)
}

type metricRegistry struct {
	metrics map[string]interface{}
	mutex   sync.RWMutex
}

func newRegistry() registry {
	return &metricRegistry{metrics: make(map[string]interface{})}
}

func (r *metricRegistry) Each(f func(string, interface{})) {
	metrics := r.registered()
	for i := range metrics {
		kv := &metrics[i]
		f(kv.name, kv.value)
	}
}

func (r *metricRegistry) Get(name string) interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.metrics[name]
}

func (r *metricRegistry) Register(name string, i interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.register(name, i)
}

func (r *metricRegistry) register(name string, i interface{}) error {
	if _, ok := r.metrics[name]; ok {
		return fmt.Errorf("duplicate metric: %s", name)
	}
	switch i.(type) {
	case *counter, groupedMetrics:
		r.metrics[name] = i
	}
	return nil
}

func (r *metricRegistry) Deregister(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.metrics, name)
}

type metricKV struct {
	name  string
	value interface{}
}

func (r *metricRegistry) registered() []metricKV {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	metrics := make([]metricKV, 0, len(r.metrics))
	for name, i := range r.metrics {
		metrics = append(metrics, metricKV{
			name:  name,
			value: i,
		})
	}
	return metrics
}
