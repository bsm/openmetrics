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
	return fmt.Sprintf("metric %q is already registered", e.Existing.Desc().FullName())
}

var defaultRegisty = NewRegistry()

// A Registry registers metric families and periodically collects their states.
type Registry struct {
	// Custom error handler, defaults to WarnOnError.
	OnError ErrorHandler

	fams []*metricFamily
	snap snapshot
	bw   bufferedWriter
	now  func() time.Time
	mu   sync.Mutex
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

// AddCounter registers a counter.
func (r *Registry) AddCounter(desc Desc) (CounterFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := counterFamily{metricFamily: metricFamily{
		desc: desc,
		mt:   CounterType,
		factory: func() (Metric, error) {
			return NewCounter(CounterOptions{CreatedAt: r.now(), OnError: r.onError()}), nil
		},
		onError: r.onError(),
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// Counter registers a counter. It panics on errors.
func (r *Registry) Counter(desc Desc) CounterFamily {
	fam, err := r.AddCounter(desc)
	if err != nil {
		panic(err)
	}
	return fam
}

// AddGauge registers a gauge.
func (r *Registry) AddGauge(desc Desc) (GaugeFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := gaugeFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      GaugeType,
		factory: func() (Metric, error) { return NewGauge(GaugeOptions{}), nil },
		onError: r.onError(),
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// Gauge registers a gauge. It panics on errors.
func (r *Registry) Gauge(desc Desc) GaugeFamily {
	fam, err := r.AddGauge(desc)
	if err != nil {
		panic(err)
	}
	return fam
}

// AddHistogram registers a histogram.
//
// The bucket boundaries for that are described
// by the bounds. Each boundary defines the upper threshold bound of a bucket.
//
// When len(bounds) is 0 the histogram will be created with a single bucket with
// an +Inf threshold.
func (r *Registry) AddHistogram(desc Desc, bounds []float64) (HistogramFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	// instant sanity check
	if err := histogramValidateBounds(bounds); err != nil {
		return nil, err
	}

	fam := histogramFamily{metricFamily: metricFamily{
		desc: desc,
		mt:   HistogramType,
		factory: func() (Metric, error) {
			return NewHistogram(bounds, HistogramOptions{CreatedAt: r.now(), OnError: r.onError()})
		},
		onError: r.onError(),
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// Histogram registers a histogram. It panics on errors.
//
// The bucket boundaries for that are described
// by the bounds. Each boundary defines the upper threshold bound of a bucket.
//
// When len(bounds) is 0 the histogram will be created with a single bucket with
// an +Inf threshold.
func (r *Registry) Histogram(desc Desc, bounds []float64) HistogramFamily {
	fam, err := r.AddHistogram(desc, bounds)
	if err != nil {
		panic(err)
	}
	return fam
}

// AddInfo registers an info.
func (r *Registry) AddInfo(desc Desc) (InfoFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := infoFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      InfoType,
		factory: func() (Metric, error) { return NewInfo(InfoOptions{}), nil },
		onError: r.onError(),
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// Info registers an info. It panics on errors.
func (r *Registry) Info(desc Desc) InfoFamily {
	fam, err := r.AddInfo(desc)
	if err != nil {
		panic(err)
	}
	return fam
}

// AddStateSet registers a state set.
func (r *Registry) AddStateSet(desc Desc, names []string) (StateSetFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := stateSetFamily{metricFamily: metricFamily{
		desc: desc,
		mt:   StateSetType,
		factory: func() (Metric, error) {
			return NewStateSet(names, StateSetOptions{OnError: r.onError()}), nil
		},
		onError: r.onError(),
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// StateSet registers a state set. It panics on errors.
func (r *Registry) StateSet(desc Desc, names []string) StateSetFamily {
	fam, err := r.AddStateSet(desc, names)
	if err != nil {
		panic(err)
	}
	return fam
}

// AddSummary registers a summary.
func (r *Registry) AddSummary(desc Desc) (SummaryFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := summaryFamily{metricFamily: metricFamily{
		desc: desc,
		mt:   SummaryType,
		factory: func() (Metric, error) {
			return NewSummary(SummaryOptions{CreatedAt: r.now(), OnError: r.onError()})
		},
		onError: r.onError(),
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// Summary registers a summary. It panics on errors.
func (r *Registry) Summary(desc Desc) SummaryFamily {
	fam, err := r.AddSummary(desc)
	if err != nil {
		panic(err)
	}
	return fam
}

// AddUnknown registers an unknown.
func (r *Registry) AddUnknown(desc Desc) (GaugeFamily, error) {
	if err := desc.Validate(); err != nil {
		return nil, err
	}

	fam := gaugeFamily{metricFamily: metricFamily{
		desc:    desc,
		mt:      UnknownType,
		factory: func() (Metric, error) { return NewGauge(GaugeOptions{}), nil },
		onError: r.onError(),
	}}
	if err := r.register(&fam.metricFamily); err != nil {
		return nil, err
	}

	return &fam, nil
}

// Unknown registers an unknown. It panics on errors.
func (r *Registry) Unknown(desc Desc) GaugeFamily {
	fam, err := r.AddUnknown(desc)
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

func (r *Registry) onError() ErrorHandler {
	if r.OnError != nil {
		return r.OnError
	}
	return WarnOnError
}
