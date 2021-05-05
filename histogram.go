package openmetrics

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

var (
	errHistogramNegative = fmt.Errorf("histograms cannot accept negative values")
	errHistogramNaN      = fmt.Errorf("histograms cannot accept NaN values")
	errHistogramInf      = fmt.Errorf("histograms cannot accept infinity values")
)

// HistogramFamily is a metric family of Histograms.
type HistogramFamily interface {
	MetricFamily

	// With returns a Histogram for the given label values.
	With(labelValues ...string) (Histogram, error)
	// Must behaves like With but panics on errors.
	Must(labelValues ...string) Histogram
}

type histogramFamily struct {
	metricFamily
}

func (f *histogramFamily) Must(labelValues ...string) Histogram {
	ist, err := f.With(labelValues...)
	if err != nil {
		panic(err)
	}
	return ist
}

func (f *histogramFamily) With(labelValues ...string) (Histogram, error) {
	ist, err := f.with(labelValues...)
	if err != nil {
		return nil, err
	}
	return ist.(Histogram), nil
}

// ----------------------------------------------------------------------------

// Histogram is an Instrument.
type Histogram interface {
	Instrument

	// Observe adds an observation.
	Observe(val float64) error
	// ObserveWithExemplarAt adds an observation with extra labels.
	// The combined length of the label names and values of MUST NOT exceed 128 UTF-8 characters.
	ObserveWithExemplar(val float64, labels LabelSet) error
	// ObserveWithExemplarAt adds an observation with extra labels at t.
	// The combined length of the label names and values of MUST NOT exceed 128 UTF-8 characters.
	ObserveWithExemplarAt(val float64, tt time.Time, labels LabelSet) error

	// MustObserve behaves like Observe but panics on errors.
	MustObserve(val float64)
	// MustObserveWithExemplarAt behaves like ObserveWithExemplarAt but panics on errors.
	MustObserveWithExemplar(val float64, labels LabelSet)
	// MustObserveWithExemplarAt behaves like ObserveWithExemplarAt but panics on errors.
	MustObserveWithExemplarAt(val float64, tt time.Time, labels LabelSet)

	// Reset resets the histogram to its original state.
	Reset()
	// ResetAt resets the histogram to its original state with created time set to t.
	ResetAt(tt time.Time)

	// Created returns the created time.
	Created() time.Time
	// Sum returns the sum of all observations.
	Sum() float64
	// Count returns the total number of observations.
	Count() int64
	// NumBuckets returns the number of threshold buckets.
	NumBuckets() int
}

type histogram struct {
	sum     float64
	count   int64
	created time.Time

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
	ox := b.exemplar
	b.exemplar = nil

	if ox != nil {
		ox.release()
	}
}

// NewHistogram inits a new value.
func NewHistogram(bounds ...float64) (Histogram, error) {
	return NewHistogramAt(time.Now(), bounds...)
}

// NewHistogramAt inits a new value at t. The bucket boundaries for that
// are described by the bounds. Each boundary defines the upper threshold bound
// of a bucket.
//
// When len(bounds) is 0 the histogram will be created with a single bucket with
// an +Inf threshold.
func NewHistogramAt(tt time.Time, bounds ...float64) (Histogram, error) {
	// validate bounds
	for i, b := range bounds {
		if math.IsNaN(b) {
			return nil, fmt.Errorf("histogram bounds must not contain NaN")
		}

		if i != 0 && b <= bounds[i-1] {
			return nil, fmt.Errorf("histogram bounds must be in strictly ascending order")
		}
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

	return &histogram{
		created: tt,
		bounds:  bounds,
		buckets: buckets,
	}, nil
}

func (t *histogram) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, b := range t.buckets {
		dst = append(dst, MetricPoint{
			Suffix:   SuffixBucket,
			Value:    float64(b.count),
			Label:    b.label,
			Exemplar: b.exemplar,
		})
	}

	return append(dst,
		MetricPoint{Suffix: SuffixCount, Value: float64(t.count)},
		MetricPoint{Suffix: SuffixSum, Value: t.sum},
		MetricPoint{Suffix: SuffixCreated, Value: asEpoch(t.created)},
	), nil
}

func (t *histogram) Observe(val float64) error {
	err := validateHistogramValue(val)
	if err != nil {
		return err
	}

	t.mu.Lock()
	t.sum += val
	t.count++
	b := &t.buckets[t.search(val)]
	b.count++
	t.mu.Unlock()
	return nil
}

func (t *histogram) ObserveWithExemplar(val float64, labels LabelSet) error {
	return t.ObserveWithExemplarAt(val, time.Time{}, labels)
}

func (t *histogram) ObserveWithExemplarAt(val float64, tt time.Time, labels LabelSet) error {
	err := validateHistogramValue(val)
	if err != nil {
		return err
	}

	var nx, ox *Exemplar
	if nx, err = poolExemplar(val, tt, labels); err != nil {
		return err
	}

	t.mu.Lock()
	t.sum += val
	t.count++
	b := &t.buckets[t.search(val)]
	b.count++
	b.exemplar, ox = nx, b.exemplar
	t.mu.Unlock()

	if ox != nil {
		ox.release()
	}
	return nil
}

func (t *histogram) MustObserve(val float64) {
	if err := t.Observe(val); err != nil {
		panic(err)
	}
}

func (t *histogram) MustObserveWithExemplar(val float64, labels LabelSet) {
	if err := t.ObserveWithExemplar(val, labels); err != nil {
		panic(err)
	}
}

func (t *histogram) MustObserveWithExemplarAt(val float64, tt time.Time, labels LabelSet) {
	if err := t.ObserveWithExemplarAt(val, tt, labels); err != nil {
		panic(err)
	}
}

func (t *histogram) Reset() {
	t.ResetAt(time.Now())
}

func (t *histogram) ResetAt(tt time.Time) {
	t.mu.Lock()
	defer t.mu.RUnlock()

	t.sum = 0
	t.count = 0
	t.created = tt

	for _, b := range t.buckets {
		b.Reset()
	}
}

func (t *histogram) Created() time.Time {
	t.mu.RLock()
	v := t.created
	t.mu.RUnlock()
	return v
}
func (t *histogram) Sum() float64 {
	t.mu.RLock()
	v := t.sum
	t.mu.RUnlock()
	return v
}
func (t *histogram) Count() int64 {
	t.mu.RLock()
	v := t.count
	t.mu.RUnlock()
	return v
}
func (t *histogram) NumBuckets() int {
	return len(t.buckets)
}

func (t *histogram) search(val float64) int {
	if len(t.bounds) > 50 {
		return sort.SearchFloat64s(t.bounds, val)
	}

	for i, b := range t.bounds {
		if val < b {
			return i
		}
	}
	return len(t.bounds)
}

func validateHistogramValue(val float64) error {
	if val < 0 {
		return errHistogramNegative
	} else if math.IsNaN(val) {
		return errHistogramNaN
	} else if math.IsInf(val, 1) {
		return errHistogramInf
	}
	return nil
}
