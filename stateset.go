package openmetrics

import (
	"fmt"
	"sync"
)

// StateSetFamily is a metric family of StateSets.
type StateSetFamily interface {
	MetricFamily

	// With returns a StateSet for the given label values.
	With(labelValues ...string) StateSet
}

type stateSetFamily struct {
	metricFamily
}

func (f *stateSetFamily) With(labelValues ...string) StateSet {
	met, err := f.with(labelValues...)
	if err != nil {
		f.onError(err)
		return nullStateSet{}
	}
	return met.(StateSet)
}

// ----------------------------------------------------------------------------

// StateSetOptions configure StateSet instances.
type StateSetOptions struct {
	OnError ErrorHandler // defaults to WarnOnError
}

// StateSet is a Metric.
type StateSet interface {
	Metric

	// Set sets a state by name.
	Set(name string, val bool)
	// Toggle toggles a state by name.
	Toggle(name string)

	// Reset resets the states.
	Reset(StateSetOptions)

	// IsEnabled returns true if a state is enabled.
	IsEnabled(name string) bool
	// Contains returns true if a state is included in the set.
	Contains(name string) bool
	// Len returns the number of states in the set.
	Len() int
}

type stateSet struct {
	names   []string
	values  []bool
	onError ErrorHandler
	mu      sync.RWMutex
}

// NewStateSet inits a new StateSet.
func NewStateSet(names []string, opts StateSetOptions) StateSet {
	unique := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, ok := seen[name]; !ok {
			unique = append(unique, name)
			seen[name] = struct{}{}
		}
	}

	m := &stateSet{names: unique, values: make([]bool, len(unique))}
	m.Reset(opts)
	return m
}

func (m *stateSet) AppendPoints(dst []MetricPoint, desc *Desc) ([]MetricPoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i, name := range m.names {
		dst = append(dst, MetricPoint{
			Label: Label{Name: desc.Name, Value: name},
			Value: m.numValue(i),
		})
	}
	return dst, nil
}

func (m *stateSet) Set(name string, enabled bool) {
	m.mu.Lock()
	pos, ok := m.search(name)
	if ok {
		m.values[pos] = enabled
	}
	m.mu.Unlock()

	if !ok {
		m.onError(fmt.Errorf("attempted to set invalid state %q", name))
	}
}

func (m *stateSet) Toggle(name string) {
	m.mu.Lock()
	pos, ok := m.search(name)
	if ok {
		m.values[pos] = !m.values[pos]
	}
	m.mu.Unlock()

	if !ok {
		m.onError(fmt.Errorf("attempted to toggle invalid state %q", name))
	}
}

func (m *stateSet) Reset(opts StateSetOptions) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.values {
		m.values[i] = false
	}

	m.onError = opts.OnError
	if m.onError == nil {
		m.onError = WarnOnError
	}
}

func (m *stateSet) IsEnabled(name string) bool {
	var v bool
	m.mu.RLock()
	if pos, ok := m.search(name); ok {
		v = m.values[pos]
	}
	m.mu.RUnlock()
	return v
}

func (m *stateSet) Contains(name string) bool {
	var v bool
	m.mu.RLock()
	_, v = m.search(name)
	m.mu.RUnlock()
	return v
}

func (m *stateSet) Len() int {
	m.mu.RLock()
	v := len(m.names)
	m.mu.RUnlock()
	return v
}

func (m *stateSet) search(name string) (int, bool) {
	for i, sn := range m.names {
		if sn == name {
			return i, true
		}
	}
	return -1, false
}

func (m *stateSet) numValue(pos int) float64 {
	if m.values[pos] {
		return 1
	}
	return 0
}

type nullStateSet struct{}

func (nullStateSet) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) { return dst, nil }

func (nullStateSet) Set(_ string, _ bool)    {}
func (nullStateSet) Toggle(_ string)         {}
func (nullStateSet) Reset(_ StateSetOptions) {}
func (nullStateSet) IsEnabled(_ string) bool { return false }
func (nullStateSet) Contains(_ string) bool  { return false }
func (nullStateSet) Len() int                { return 0 }
