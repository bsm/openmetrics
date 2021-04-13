package openmetrics

import (
	"math"
	"sync/atomic"
)

// GaugeFamily is a metric family of Gauges.
type GaugeFamily interface {
	MetricFamily

	// With returns a Gauge for the given label values.
	With(labelValues ...string) (Gauge, error)
	// Must behaves like With but panics on errors.
	Must(labelValues ...string) Gauge
}

type gaugeFamily struct {
	metricFamily
}

func (f *gaugeFamily) Must(labelValues ...string) Gauge {
	ist, err := f.With(labelValues...)
	if err != nil {
		panic(err)
	}
	return ist
}

func (f *gaugeFamily) With(labelValues ...string) (Gauge, error) {
	ist, err := f.with(labelValues...)
	if err != nil {
		return nil, err
	}
	return ist.(Gauge), nil
}

// ----------------------------------------------------------------------------

const nav = 0x7FF9000000000001

// Gauge is an Instrument.
type Gauge interface {
	Instrument

	// Set sets the value.
	Set(val float64)
	// Add increments the value.
	Add(val float64)
	// Value returns the current value.
	Value() float64
	// Reset resets the gauge to its original state.
	Reset()
}

type gauge uint64

// NewGauge inits a new Gauge.
func NewGauge() Gauge {
	var v gauge = nav
	return &v
}

func (t *gauge) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	if *t == nav {
		return dst, nil
	}

	return append(dst,
		MetricPoint{Value: t.Value()},
	), nil
}

func (t *gauge) Set(val float64) {
	atomic.StoreUint64((*uint64)(t), math.Float64bits(val))
}

func (t *gauge) Add(val float64) {
	for {
		cur := atomic.LoadUint64((*uint64)(t))
		upd := math.Float64bits(math.Float64frombits(cur) + val)
		if atomic.CompareAndSwapUint64((*uint64)(t), cur, upd) {
			return
		}
	}
}

func (t *gauge) Reset() {
	atomic.StoreUint64((*uint64)(t), nav)
}

func (t *gauge) Value() float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(t)))
}
