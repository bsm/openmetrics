package openmetrics

import (
	"fmt"
	"math"
	"sync"
	"time"
)

var (
	errCounterNegative = fmt.Errorf("counters must be monotonically non-decreasing")
	errCounterNaN      = fmt.Errorf("counters cannot accept NaN values")
	errCounterInf      = fmt.Errorf("counters cannot accept infinity values")
)

// CounterFamily is a metric family of counters.
type CounterFamily interface {
	MetricFamily

	// With returns a counter for the given label values.
	With(labelValues ...string) (Counter, error)
	// Must behaves like With but panics on errors.
	Must(labelValues ...string) Counter
}

type counterFamily struct {
	metricFamily
}

func (f *counterFamily) Must(labelValues ...string) Counter {
	ist, err := f.With(labelValues...)
	if err != nil {
		panic(err)
	}
	return ist
}

func (f *counterFamily) With(labelValues ...string) (Counter, error) {
	ist, err := f.with(labelValues...)
	if err != nil {
		return nil, err
	}
	return ist.(Counter), nil
}

// ----------------------------------------------------------------------------

// Counter is an Instrument.
type Counter interface {
	Instrument

	// Add increments the total. Total MUST be monotonically
	// non-decreasing over time.
	Add(val float64) error
	// AddWithExemplarAt adds a value with extra labels.
	// The combined length of the label names and values of MUST NOT exceed 128 UTF-8 characters.
	AddWithExemplar(val float64, labels LabelSet) error
	// AddWithExemplarAt adds a value with extra labels at tt.
	// The combined length of the label names and values of MUST NOT exceed 128 UTF-8 characters.
	AddWithExemplarAt(val float64, tt time.Time, labels LabelSet) error

	// MustAdd behaves like Add but panics on invalid inputs.
	MustAdd(val float64)
	// MustAddWithExemplar behaves like AddWithExemplar but panics on invalid inputs.
	MustAddWithExemplar(val float64, labels LabelSet)
	// MustAddWithExemplarAt behaves like AddWithExemplarAt but panics on invalid inputs.
	MustAddWithExemplarAt(val float64, tt time.Time, labels LabelSet)

	// Reset resets the created time to now and the total to 0.
	Reset()
	// ResetAt resets the created time to t and the total to 0.
	ResetAt(t time.Time)

	// Created returns the created time.
	Created() time.Time
	// Total returns the current total.
	Total() float64
	// Exemplar returns the most recent Exemplar.
	Exemplar() *Exemplar
}

type counter struct {
	total    float64
	created  time.Time
	exemplar *Exemplar

	mu sync.RWMutex
}

// NewCounter inits a new counter.
func NewCounter() Counter {
	return NewCounterAt(time.Now())
}

// NewCounterAt inits a new value at t.
func NewCounterAt(t time.Time) Counter {
	return &counter{created: t}
}

func (t *counter) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return append(dst,
		MetricPoint{
			Suffix:   SuffixTotal,
			Value:    t.total,
			Exemplar: t.exemplar,
		},
		MetricPoint{
			Suffix: SuffixCreated,
			Value:  asEpoch(t.created),
		},
	), nil
}

func (t *counter) Add(val float64) error {
	if err := validateCounterValue(val); err != nil {
		return err
	}

	t.mu.Lock()
	t.total += val
	t.mu.Unlock()
	return nil
}

func (t *counter) AddWithExemplar(val float64, labels LabelSet) error {
	return t.AddWithExemplarAt(val, time.Time{}, labels)
}

func (t *counter) AddWithExemplarAt(val float64, tt time.Time, labels LabelSet) error {
	err := validateCounterValue(val)
	if err != nil {
		return err
	}

	var nx, ox *Exemplar
	if nx, err = poolExemplar(val, tt, labels); err != nil {
		return err
	}

	t.mu.Lock()
	t.total += val
	ox, t.exemplar = t.exemplar, nx
	t.mu.Unlock()

	if ox != nil {
		ox.release()
	}
	return nil
}

func (t *counter) MustAdd(val float64) {
	if err := t.Add(val); err != nil {
		panic(err)
	}
}

func (t *counter) MustAddWithExemplar(val float64, labels LabelSet) {
	if err := t.AddWithExemplar(val, labels); err != nil {
		panic(err)
	}
}

func (t *counter) MustAddWithExemplarAt(val float64, tt time.Time, labels LabelSet) {
	if err := t.AddWithExemplarAt(val, tt, labels); err != nil {
		panic(err)
	}
}

func (t *counter) Reset() {
	t.ResetAt(time.Now())
}

func (t *counter) ResetAt(now time.Time) {
	t.mu.Lock()
	old := t.exemplar
	t.total = 0
	t.created = now
	t.exemplar = nil
	t.mu.Unlock()

	if old != nil {
		old.release()
	}
}

func (t *counter) Created() time.Time {
	t.mu.RLock()
	v := t.created
	t.mu.RUnlock()
	return v
}

func (t *counter) Total() float64 {
	t.mu.RLock()
	v := t.total
	t.mu.RUnlock()
	return v
}

func (t *counter) Exemplar() *Exemplar {
	t.mu.RLock()
	x := t.exemplar
	t.mu.RUnlock()
	return x
}

func validateCounterValue(val float64) error {
	if val < 0 {
		return errCounterNegative
	} else if math.IsNaN(val) {
		return errCounterNaN
	} else if math.IsInf(val, 1) {
		return errCounterInf
	}
	return nil
}
