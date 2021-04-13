package openmetrics

import (
	"bufio"
	"fmt"
	"io"
	"sync"
	"time"
)

// ErrAlreadyRegistered is returned when a metric is already registered.
type ErrAlreadyRegistered struct {
	Existing MetricFamily
}

func (e ErrAlreadyRegistered) Error() string {
	return fmt.Sprintf("metric %q is already registered", e.Existing.Desc().fullName())
}

var defaultRegisty = NewRegistry()

// A Registry registers metric families and periodically collects their states.
type Registry struct {
	fams []*metricFamily

	snap snapshot
	bw   bufferedWriter
	now  func() time.Time

	mu sync.Mutex
}

// DefaultRegistry returns the default registry instance.
func DefaultRegistry() *Registry {
	return defaultRegisty
}

// NewRegistry returns a new registry instance.
func NewRegistry() *Registry {
	return &Registry{
		now: time.Now,
	}
}

// NewConsistentRegistry created a new registry instance that uses a custom time
// function and produces consistent outputs. This is useful for tests.
func NewConsistentRegistry(now func() time.Time) *Registry {
	reg := NewRegistry()
	reg.now = now
	reg.snap.cos = uint64Slice{}
	return reg
}

// Counter registers a counter.
func (r *Registry) Counter(desc Desc) (CounterFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := counterFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      CounterType,
		factory: func() (Instrument, error) { return NewCounterAt(r.now()), nil },
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// MustCounter registers a counter. It panics on errors.
func (r *Registry) MustCounter(desc Desc) CounterFamily {
	fam, err := r.Counter(desc)
	if err != nil {
		panic(err)
	}
	return fam
}

// Gauge registers a gauge.
func (r *Registry) Gauge(desc Desc) (GaugeFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := gaugeFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      GaugeType,
		factory: func() (Instrument, error) { return NewGauge(), nil },
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// MustGauge registers a gauge. It panics on errors.
func (r *Registry) MustGauge(desc Desc) GaugeFamily {
	fam, err := r.Gauge(desc)
	if err != nil {
		panic(err)
	}
	return fam
}

// Histogram registers a histogram.
//
// The bucket boundaries for that are described
// by the bounds. Each boundary defines the upper threshold bound of a bucket.
//
// When len(bounds) is 0 the histogram will be created with a single bucket with
// an +Inf threshold.
func (r *Registry) Histogram(desc Desc, bounds ...float64) (HistogramFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := histogramFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      HistogramType,
		factory: func() (Instrument, error) { return NewHistogramAt(r.now(), bounds...) },
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// MustHistogram registers a histogram. It panics on errors.
//
// The bucket boundaries for that are described
// by the bounds. Each boundary defines the upper threshold bound of a bucket.
//
// When len(bounds) is 0 the histogram will be created with a single bucket with
// an +Inf threshold.
func (r *Registry) MustHistogram(desc Desc, bounds ...float64) HistogramFamily {
	fam, err := r.Histogram(desc, bounds...)
	if err != nil {
		panic(err)
	}
	return fam
}

// Info registers an info.
func (r *Registry) Info(desc Desc) (InfoFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := infoFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      InfoType,
		factory: func() (Instrument, error) { return NewInfo(), nil },
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// MustInfo registers an info. It panics on errors.
func (r *Registry) MustInfo(desc Desc) InfoFamily {
	fam, err := r.Info(desc)
	if err != nil {
		panic(err)
	}
	return fam
}

// StateSet registers a state set.
func (r *Registry) StateSet(desc Desc, names ...string) (StateSetFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := stateSetFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      StateSetType,
		factory: func() (Instrument, error) { return NewStateSet(names...), nil },
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// MustStateSet registers a state set. It panics on errors.
func (r *Registry) MustStateSet(desc Desc, names ...string) StateSetFamily {
	fam, err := r.StateSet(desc, names...)
	if err != nil {
		panic(err)
	}
	return fam
}

// Unknown registers an unknown.
func (r *Registry) Unknown(desc Desc) (GaugeFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := gaugeFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      UnknownType,
		factory: func() (Instrument, error) { return NewGauge(), nil },
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// MustUnknown registers an unknown. It panics on errors.
func (r *Registry) MustUnknown(desc Desc) GaugeFamily {
	fam, err := r.Unknown(desc)
	if err != nil {
		panic(err)
	}
	return fam
}

// WriteTo implements io.WriterTo interface.
func (r *Registry) WriteTo(w io.Writer) (int64, error) {
	var total int64

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.bw.Writer == nil {
		r.bw.Writer = bufio.NewWriter(w)
	} else {
		r.bw.Reset(w)
	}

	for _, fam := range r.fams {
		err := fam.snapshot(&r.snap)
		if err != nil {
			return total, err
		}

		nn, err := r.snap.WriteTo(&r.bw)
		total += nn
		if err != nil {
			return total, err
		}
	}

	n, err := r.bw.WriteString("# EOF\n")
	total += int64(n)
	if err != nil {
		return total, err
	}

	if err := r.bw.Flush(); err != nil {
		return total, err
	}

	return total, nil
}

func (r *Registry) register(fam *metricFamily) error {
	uid := fam.desc.calcID()

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, fam := range r.fams {
		if fam.ID() == uid {
			return ErrAlreadyRegistered{Existing: fam}
		}
	}

	r.fams = append(r.fams, fam)
	return nil
}
