package openmetrics

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

// HistogramFamily is a metric family of Histograms.
type HistogramFamily interface {
	MetricFamily

	// With returns a Histogram for the given label values.
	With(labelValues ...string) Histogram
}

type histogramFamily struct {
	metricFamily
}

func (f *histogramFamily) With(labelValues ...string) Histogram {
	met, err := f.with(labelValues...)
	if err != nil {
		f.onError(err)
		return nullHistogram{}
	}
	return met.(Histogram)
}

// ----------------------------------------------------------------------------

// HistogramOptions configure Histogram instances.
type HistogramOptions struct {
	CreatedAt time.Time    // defaults to time.Now()
	OnError   ErrorHandler // defaults to WarnOnError
}

// Histogram is a Metric.
type Histogram interface {
	Metric

	// Observe adds an observation. Attempts to pass negative, NaN or infinity
	// values will result in a panic.
	Observe(float64)

	// ObserveExemplar adds an observation using an exemplar. Attempts to pass
	// negative, NaN or infinity values will result in a panic. Invalid exemplars
	// will be silently discarded.
	ObserveExemplar(*Exemplar)

	// Reset resets the histogram to its original state.
	Reset(HistogramOptions)

	// Created returns the created time.
	Created() time.Time
	// Sum returns the sum of all observations.
	Sum() float64
	// Count returns the total number of observations.
	Count() int64
	// NumBuckets returns the number of threshold buckets.
	NumBuckets() int
	// Exemplar returns the exemplar at bucket index.
	Exemplar(bucket int) *Exemplar
}

type histogram struct {
	sum     float64
	count   int64
	created time.Time
	onError ErrorHandler

	bounds  []float64
	buckets []histogramBucket

	mu sync.RWMutex
}

type histogramBucket struct {
	count    int64
	exemplar *Exemplar
	label    Label
}

func (b *histogramBucket) Reset() {
	b.count = 0
	b.exemplar = nil
}

// NewHistogram inits a new histogram. The bucket boundaries for that are
// described by the bounds. Each boundary defines the upper threshold bound of a
// bucket.
//
// When len(bounds) is 0 the histogram will be created with a single bucket with
// an +Inf threshold.
func NewHistogram(bounds []float64, opts HistogramOptions) (Histogram, error) {
	if err := histogramValidateBounds(bounds); err != nil {
		return nil, err
	}

	// trim bounds
	if n := len(bounds); n != 0 && math.IsInf(bounds[n-1], 1) {
		bounds = bounds[:n-1]
	}

	// create buckets
	buckets := make([]histogramBucket, len(bounds)+1)
	for i, b := range bounds {
		buckets[i].label = Label{Name: "le", Value: strconv.FormatFloat(b, 'g', -1, 64)}
	}
	buckets[len(bounds)].label = Label{Name: "le", Value: "+Inf"}

	m := &histogram{
		bounds:  bounds,
		buckets: buckets,
	}
	m.Reset(opts)
	return m, nil
}

func (m *histogram) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, b := range m.buckets {
		dst = append(dst, MetricPoint{
			Suffix:   SuffixBucket,
			Value:    float64(b.count),
			Label:    b.label,
			Exemplar: b.exemplar,
		})
	}

	return append(dst,
		MetricPoint{Suffix: SuffixCount, Value: float64(m.count)},
		MetricPoint{Suffix: SuffixSum, Value: m.sum},
		MetricPoint{Suffix: SuffixCreated, Value: asEpoch(m.created)},
	), nil
}

func (m *histogram) Observe(val float64) {
	if err := histogramValidateValue(val); err != nil {
		m.handleError(err)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.sum += val
	m.count++
	b := &m.buckets[m.search(val)]
	b.count++
}

func (m *histogram) ObserveExemplar(ex *Exemplar) {
	if err := histogramValidateValue(ex.Value); err != nil {
		m.handleError(err)
		return
	}

	if err := ex.Validate(); err != nil {
		m.handleError(err)
		m.Observe(ex.Value)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.sum += ex.Value
	m.count++
	b := &m.buckets[m.search(ex.Value)]
	b.count++
	if b.exemplar == nil {
		b.exemplar = new(Exemplar)
	}
	b.exemplar.copyFrom(ex)
}

func (m *histogram) Reset(opts HistogramOptions) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sum = 0
	m.count = 0
	m.created = opts.CreatedAt
	m.onError = opts.OnError
	for _, b := range m.buckets {
		b.Reset()
	}

	if m.created.IsZero() {
		m.created = time.Now()
	}
	if m.onError == nil {
		m.onError = WarnOnError
	}
}

func (m *histogram) Created() time.Time {
	m.mu.RLock()
	v := m.created
	m.mu.RUnlock()
	return v
}

func (m *histogram) Sum() float64 {
	m.mu.RLock()
	v := m.sum
	m.mu.RUnlock()
	return v
}

func (m *histogram) Count() int64 {
	m.mu.RLock()
	v := m.count
	m.mu.RUnlock()
	return v
}

func (m *histogram) NumBuckets() int {
	return len(m.buckets)
}

func (m *histogram) Exemplar(n int) *Exemplar {
	if n < 0 || n >= len(m.buckets) {
		return nil
	}

	m.mu.RLock()
	v := m.buckets[n].exemplar
	m.mu.RUnlock()
	return v
}

func (m *histogram) search(val float64) int {
	if len(m.bounds) > 50 {
		return sort.SearchFloat64s(m.bounds, val)
	}

	for i, b := range m.bounds {
		if val < b {
			return i
		}
	}
	return len(m.bounds)
}

func (m *histogram) handleError(err error) {
	m.mu.RLock()
	m.onError(err)
	m.mu.RUnlock()
}

type nullHistogram struct{}

func (nullHistogram) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) { return dst, nil }

func (nullHistogram) Observe(_ float64)           {}
func (nullHistogram) ObserveExemplar(_ *Exemplar) {}
func (nullHistogram) Reset(_ HistogramOptions)    {}
func (nullHistogram) Created() time.Time          { return time.Time{} }
func (nullHistogram) Sum() float64                { return 0.0 }
func (nullHistogram) Count() int64                { return 0 }
func (nullHistogram) NumBuckets() int             { return 1 }
func (nullHistogram) Exemplar(_ int) *Exemplar    { return nil }

var (
	errHistogramBoundsNaN    = fmt.Errorf("histogram bounds must not contain NaN")
	errHistogramBoundsNonAsc = fmt.Errorf("histogram bounds must be in strictly ascending order")

	errHistogramValNegative = fmt.Errorf("histograms cannot accept negative values")
	errHistogramValNaN      = fmt.Errorf("histograms cannot accept NaN values")
	errHistogramValInf      = fmt.Errorf("histograms cannot accept infinity values")
)

func histogramValidateBounds(bounds []float64) error {
	for i, b := range bounds {
		if math.IsNaN(b) {
			return errHistogramBoundsNaN
		}

		if i != 0 && b <= bounds[i-1] {
			return errHistogramBoundsNonAsc
		}
	}
	return nil
}

func histogramValidateValue(val float64) error {
	if val < 0 {
		return errHistogramValNegative
	} else if math.IsNaN(val) {
		return errHistogramValNaN
	} else if math.IsInf(val, 1) {
		return errHistogramValInf
	}
	return nil
}
