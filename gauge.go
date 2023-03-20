package openmetrics

import (
	"math"
	"sync/atomic"
)

// GaugeFamily is a metric family of Gauges.
type GaugeFamily interface {
	MetricFamily

	// With returns a Gauge for the given label values.
	With(labelValues ...string) Gauge
}

type gaugeFamily struct {
	metricFamily
}

func (f *gaugeFamily) With(labelValues ...string) Gauge {
	met, err := f.with(labelValues...)
	if err != nil {
		f.onError(err)
		return nullGauge{}
	}
	return met.(Gauge)
}

// ----------------------------------------------------------------------------

const float64BitsNaN = 0x7FF9000000000001

// GaugeOptions configure Gauge instances.
type GaugeOptions struct{}

// Gauge is a Metric.
type Gauge interface {
	Metric

	// Set sets the value.
	Set(val float64)
	// Add increments the value.
	Add(val float64)
	// Value returns the current value.
	Value() float64
	// Reset resets the gauge to its original state.
	Reset(GaugeOptions)
}

type gauge uint64

// NewGauge inits a new Gauge.
func NewGauge(_ GaugeOptions) Gauge {
	var v gauge = float64BitsNaN
	return &v
}

func (m *gauge) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	if *m == float64BitsNaN {
		return dst, nil
	}

	return append(dst,
		MetricPoint{Value: m.Value()},
	), nil
}

func (m *gauge) Set(val float64) {
	atomic.StoreUint64((*uint64)(m), math.Float64bits(val))
}

func (m *gauge) Add(val float64) {
	for {
		curBits := atomic.LoadUint64((*uint64)(m))

		// if current gauge value is NaN - assume zero
		var curFloat float64 = 0
		if curBits != float64BitsNaN {
			curFloat = math.Float64frombits(curBits)
		}

		sumFloat := curFloat + val
		sumBits := math.Float64bits(sumFloat)
		if atomic.CompareAndSwapUint64((*uint64)(m), curBits, sumBits) {
			return
		}
	}
}

func (m *gauge) Reset(_ GaugeOptions) {
	atomic.StoreUint64((*uint64)(m), float64BitsNaN)
}

func (m *gauge) Value() float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(m)))
}

type nullGauge struct{}

func (nullGauge) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) { return dst, nil }

func (nullGauge) Set(_ float64)        {}
func (nullGauge) Add(_ float64)        {}
func (nullGauge) Value() float64       { return 0.0 }
func (nullGauge) Reset(_ GaugeOptions) {}
