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
	With(labelValues ...string) (StateSet, error)
	// Must behaves like With but panics on errors.
	Must(labelValues ...string) StateSet
}

type stateSetFamily struct {
	metricFamily
}

func (f *stateSetFamily) Must(labelValues ...string) StateSet {
	ist, err := f.With(labelValues...)
	if err != nil {
		panic(err)
	}
	return ist
}

func (f *stateSetFamily) With(labelValues ...string) (StateSet, error) {
	ist, err := f.with(labelValues...)
	if err != nil {
		return nil, err
	}
	return ist.(StateSet), nil
}

// ----------------------------------------------------------------------------

// StateSet is an Instrument.
type StateSet interface {
	Instrument

	// Set sets a state by name.
	// Trying to set an invalid state will result in a panic.
	Set(name string, val bool) error
	// Toggle toggles a state by name.
	// Trying to toggle an invalid state will result in a panic.
	Toggle(name string) error

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
	states stateSetStateSlice
	mu     sync.RWMutex
}

// NewStateSet inits a new StateSet.
func NewStateSet(names ...string) StateSet {
	states := make(stateSetStateSlice, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, ok := seen[name]; !ok {
			states = append(states, stateSetState{Name: name})
			seen[name] = struct{}{}
		}
	}
	return &stateSet{states: states}
}

func (t *stateSet) AppendPoints(dst []MetricPoint, desc *Desc) ([]MetricPoint, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, s := range t.states {
		dst = append(dst, MetricPoint{
			Label: Label{Name: desc.Name, Value: s.Name},
			Value: s.NumValue(),
		})
	}
	return dst, nil
}

func (t *stateSet) Set(name string, enabled bool) error {
	t.mu.Lock()
	pos, ok := t.search(name)
	if ok {
		t.states[pos].Enabled = enabled
	}
	t.mu.Unlock()

	if !ok {
		return fmt.Errorf("invalid state %q", name)
	}
	return nil
}

func (t *stateSet) Toggle(name string) error {
	t.mu.Lock()
	pos, ok := t.search(name)
	if ok {
		t.states[pos].Enabled = !t.states[pos].Enabled
	}
	t.mu.Unlock()

	if !ok {
		return fmt.Errorf("invalid state %q", name)
	}
	return nil
}

func (t *stateSet) IsEnabled(name string) bool {
	var v bool
	t.mu.RLock()
	if pos, ok := t.search(name); ok {
		v = t.states[pos].Enabled
	}
	t.mu.RUnlock()
	return v
}

func (t *stateSet) Contains(name string) bool {
	var v bool
	t.mu.RLock()
	_, v = t.search(name)
	t.mu.RUnlock()
	return v
}

func (t *stateSet) Len() int {
	t.mu.RLock()
	v := len(t.states)
	t.mu.RUnlock()
	return v
}

func (t *stateSet) search(name string) (int, bool) {
	sl := t.states
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
