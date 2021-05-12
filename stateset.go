package openmetrics

import (
	"fmt"
	"sort"
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

	// IsEnabled returns true if a state is enabled.
	IsEnabled(name string) bool
	// Contains returns true if a state is included in the set.
	Contains(name string) bool
	// Len returns the number of states in the set.
	Len() int
}

type stateSetState struct {
	Name    string
	Enabled bool
}

func (s stateSetState) NumValue() float64 {
	if s.Enabled {
		return 1
	}
	return 0
}

type stateSetStateSlice []stateSetState

func (s stateSetStateSlice) Len() int           { return len(s) }
func (s stateSetStateSlice) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s stateSetStateSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type stateSet struct {
	states  stateSetStateSlice
	onError ErrorHandler
	mu      sync.RWMutex
}

// NewStateSet inits a new StateSet.
func NewStateSet(names []string, opts StateSetOptions) StateSet {
	states := make(stateSetStateSlice, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, ok := seen[name]; !ok {
			states = append(states, stateSetState{Name: name})
			seen[name] = struct{}{}
		}
	}

	onError := opts.OnError
	if onError == nil {
		onError = WarnOnError
	}

	return &stateSet{states: states, onError: onError}
}

func (m *stateSet) AppendPoints(dst []MetricPoint, desc *Desc) ([]MetricPoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.states {
		dst = append(dst, MetricPoint{
			Label: Label{Name: desc.Name, Value: s.Name},
			Value: s.NumValue(),
		})
	}
	return dst, nil
}

func (m *stateSet) Set(name string, enabled bool) {
	m.mu.Lock()
	pos, ok := m.search(name)
	if ok {
		m.states[pos].Enabled = enabled
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
		m.states[pos].Enabled = !m.states[pos].Enabled
	}
	m.mu.Unlock()

	if !ok {
		m.onError(fmt.Errorf("attempted to toggle invalid state %q", name))
	}
}

func (m *stateSet) IsEnabled(name string) bool {
	var v bool
	m.mu.RLock()
	if pos, ok := m.search(name); ok {
		v = m.states[pos].Enabled
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
	v := len(m.states)
	m.mu.RUnlock()
	return v
}

func (m *stateSet) search(name string) (int, bool) {
	sl := m.states
	if len(sl) > 20 {
		pos := sort.Search(len(sl), func(i int) bool { return sl[i].Name >= name })
		if pos < len(sl) && sl[pos].Name == name {
			return pos, true
		}
		return -1, false
	}

	for i, s := range sl {
		if s.Name == name {
			return i, true
		}
	}
	return -1, false
}

type nullStateSet struct{}

func (nullStateSet) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) { return dst, nil }

func (nullStateSet) Set(_ string, _ bool)    {}
func (nullStateSet) Toggle(_ string)         {}
func (nullStateSet) IsEnabled(_ string) bool { return false }
func (nullStateSet) Contains(_ string) bool  { return false }
func (nullStateSet) Len() int                { return 0 }
