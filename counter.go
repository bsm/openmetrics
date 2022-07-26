package openmetrics

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// CounterFamily is a metric family of counters.
type CounterFamily interface {
	MetricFamily

	// With returns a Counter for the given label values.
	With(labelValues ...string) Counter
}

type counterFamily struct {
	metricFamily
}

func (f *counterFamily) With(labelValues ...string) Counter {
	met, err := f.with(labelValues...)
	if err != nil {
		f.onError(err)
		return nullCounter{}
	}
	return met.(Counter)
}

// ----------------------------------------------------------------------------

// CounterOptions configure Counter instances.
type CounterOptions struct {
	CreatedAt time.Time    // defaults to time.Now()
	OnError   ErrorHandler // defaults to WarnOnError
}

// Counter is an Metric.
type Counter interface {
	Metric

	// Add increments the total. Total MUST be monotonically non-decreasing over
	// time. Attempts to pass negative, NaN or infinity values will result in
	// errors.
	Add(val float64)

	// AddExemplar increments the total using an exemplar. Attempts to pass
	// negative, NaN or infinity values will result in a errors. Invalid exemplars
	// will be silently discarded.
	AddExemplar(ex *Exemplar)

	// Reset resets the created time to now and the total to 0.
	Reset(CounterOptions)

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
	onError  ErrorHandler

	mu sync.RWMutex
}

// NewCounter inits a new counter.
func NewCounter(opts CounterOptions) Counter {
	m := &counter{}
	m.Reset(opts)
	return m
}

func (m *counter) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append(dst,
		MetricPoint{
			Suffix:   SuffixTotal,
			Value:    m.total,
			Exemplar: m.exemplar,
		},
		MetricPoint{
			Suffix: SuffixCreated,
			Value:  asEpoch(m.created),
		},
	), nil
}

func (m *counter) Add(val float64) {
	if err := counterValidateValue(val); err != nil {
		m.handleError(err)
		return
	}

	m.mu.Lock()
	m.total += val
	m.mu.Unlock()
}

func (m *counter) AddExemplar(ex *Exemplar) {
	if err := counterValidateValue(ex.Value); err != nil {
		m.handleError(err)
		return
	}

	if err := ex.Validate(); err != nil {
		m.handleError(err)
		m.Add(ex.Value)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.exemplar == nil {
		m.exemplar = new(Exemplar)
	}
	m.exemplar.copyFrom(ex)
	m.total += ex.Value
}

func (m *counter) Reset(opts CounterOptions) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.total = 0
	m.created = opts.CreatedAt
	m.onError = opts.OnError
	m.exemplar = nil

	if m.created.IsZero() {
		m.created = time.Now()
	}
	if m.onError == nil {
		m.onError = WarnOnError
	}
}

func (m *counter) Created() time.Time {
	m.mu.RLock()
	v := m.created
	m.mu.RUnlock()
	return v
}

func (m *counter) Total() float64 {
	m.mu.RLock()
	v := m.total
	m.mu.RUnlock()
	return v
}

func (m *counter) Exemplar() *Exemplar {
	m.mu.RLock()
	x := m.exemplar
	m.mu.RUnlock()
	return x
}

func (m *counter) handleError(err error) {
	m.mu.RLock()
	m.onError(err)
	m.mu.RUnlock()
}

type nullCounter struct{}

func (nullCounter) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) { return dst, nil }

func (nullCounter) Add(_ float64)           {}
func (nullCounter) AddExemplar(_ *Exemplar) {}
func (nullCounter) Reset(_ CounterOptions)  {}
func (nullCounter) Created() time.Time      { return time.Time{} }
func (nullCounter) Total() float64          { return 0.0 }
func (nullCounter) Exemplar() *Exemplar     { return nil }

var (
	errCounterNegative = fmt.Errorf("counters must be monotonically non-decreasing")
	errCounterNaN      = fmt.Errorf("counters cannot accept NaN values")
	errCounterInf      = fmt.Errorf("counters cannot accept infinity values")
)

func counterValidateValue(val float64) error {
	if val < 0 {
		return errCounterNegative
	} else if math.IsNaN(val) {
		return errCounterNaN
	} else if math.IsInf(val, 1) {
		return errCounterInf
	}
	return nil
}
