package openmetrics

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// SummaryFamily is a metric family of summaries.
type SummaryFamily interface {
	MetricFamily

	// With returns a Summary for the given label values.
	With(labelValues ...string) Summary
}

type summaryFamily struct {
	metricFamily
}

func (f *summaryFamily) With(labelValues ...string) Summary {
	met, err := f.with(labelValues...)
	if err != nil {
		f.onError(err)
		return nullSummary{}
	}
	return met.(Summary)
}

// ----------------------------------------------------------------------------

// SummaryOptions configure Summary instances.
type SummaryOptions struct {
	CreatedAt time.Time    // defaults to time.Now()
	OnError   ErrorHandler // defaults to WarnOnError
}

// Summary is an Metric.
type Summary interface {
	Metric

	// Observe adds an observation. Attempts to pass negative, NaN or infinity
	// values will result in an error.
	Observe(val float64)
	// Sum returns the sum of all observations.
	Sum() float64
	// Count returns the total number of observations.
	Count() int64
	// Created returns the created time.
	Created() time.Time

	// Reset resets the created time to now and the total to 0.
	Reset(SummaryOptions)
}

type summary struct {
	sum     float64
	count   int64
	created time.Time
	onError ErrorHandler

	mu sync.RWMutex
}

// NewSummary inits a new summary.
func NewSummary(opts SummaryOptions) (Summary, error) {
	// errors cannot happen here but we reserve the "right" to return some to
	// avoid backwards compatibility issues in the future

	m := &summary{}
	m.Reset(opts)
	return m, nil
}

func (m *summary) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append(dst,
		MetricPoint{
			Suffix: SuffixCount,
			Value:  float64(m.count),
		},
		MetricPoint{
			Suffix: SuffixSum,
			Value:  m.sum,
		},
		MetricPoint{
			Suffix: SuffixCreated,
			Value:  asEpoch(m.created),
		},
	), nil
}

func (m *summary) Observe(val float64) {
	if err := summaryValidateValue(val); err != nil {
		m.handleError(err)
		return
	}

	m.mu.Lock()
	m.count++
	m.sum += val
	m.mu.Unlock()
}

func (m *summary) Reset(opts SummaryOptions) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sum = 0
	m.count = 0
	m.created = opts.CreatedAt
	m.onError = opts.OnError

	if m.created.IsZero() {
		m.created = time.Now()
	}
	if m.onError == nil {
		m.onError = WarnOnError
	}
}

func (m *summary) Created() time.Time {
	m.mu.RLock()
	v := m.created
	m.mu.RUnlock()
	return v
}

func (m *summary) Sum() float64 {
	m.mu.RLock()
	v := m.sum
	m.mu.RUnlock()
	return v
}

func (m *summary) Count() int64 {
	m.mu.RLock()
	v := m.count
	m.mu.RUnlock()
	return v
}

func (m *summary) handleError(err error) {
	m.mu.RLock()
	m.onError(err)
	m.mu.RUnlock()
}

type nullSummary struct{}

func (nullSummary) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) { return dst, nil }

func (nullSummary) Observe(_ float64)      {}
func (nullSummary) Reset(_ SummaryOptions) {}
func (nullSummary) Created() time.Time     { return time.Time{} }
func (nullSummary) Sum() float64           { return 0.0 }
func (nullSummary) Count() int64           { return 0 }

var (
	errSummaryNegative = fmt.Errorf("summaries cannot accept negative values")
	errSummaryNaN      = fmt.Errorf("summaries cannot accept NaN values")
	errSummaryInf      = fmt.Errorf("summaries cannot accept infinity values")
)

func summaryValidateValue(val float64) error {
	if val < 0 {
		return errSummaryNegative
	} else if math.IsNaN(val) {
		return errSummaryNaN
	} else if math.IsInf(val, 1) {
		return errSummaryInf
	}
	return nil
}
