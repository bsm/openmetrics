package openmetrics

import (
	"sort"
	"sync"
)

// ContentType is the official content type of an openmetrics document.
const ContentType = "application/openmetrics-text; version=1.0.0; charset=utf-8"

// MetricPoint is a point in a Metric.
type MetricPoint struct {
	Suffix   MetricSuffix
	Value    float64
	Label    Label
	Exemplar *Exemplar
}

// MetricType defines the type of a Metric.
type MetricType uint8

func (t MetricType) String() string {
	switch t {
	case GaugeType:
		return "gauge"
	case CounterType:
		return "counter"
	case StateSetType:
		return "stateset"
	case InfoType:
		return "info"
	case HistogramType:
		return "histogram"
	case _GaugeHistogramType:
		return "gaugehistogram"
	case _SummaryType:
		return "summary"
	default:
		return "unknown"
	}
}

const (
	// UnknownType must use unknown MetricPoint values.
	UnknownType MetricType = iota
	// GaugeType must use gauge MetricPoint values.
	GaugeType
	// Counter must use counter MetricPoint values.
	CounterType
	// StateSetType set must use stateset MetricPoint values.
	StateSetType
	// InfoType must use info MetricPoint values.
	InfoType
	// HistogramType must use histogram MetricPoint values.
	HistogramType
	// GaugeHistogramType must use gaugehistogram value MetricPoint values.
	_GaugeHistogramType
	// Summary quantiles must use summary value MetricPoint values.
	_SummaryType
)

// MetricSuffix defines the metric suffix value.
type MetricSuffix uint8

// String returns the suffix.
func (m MetricSuffix) String() string {
	switch m {
	case SuffixTotal:
		return "_total"
	case SuffixCreated:
		return "_created"
	case SuffixCount:
		return "_count"
	case SuffixSum:
		return "_sum"
	case SuffixBucket:
		return "_bucket"
	case _SuffixGCount:
		return "_gcount"
	case _SuffixGSum:
		return "_gsum"
	case SuffixInfo:
		return "_info"
	default:
		return ""
	}
}

// MetricSuffix enum.
const (
	SuffixEmpty   MetricSuffix = iota // gauge, stateset, unknown, summary
	SuffixTotal                       // counter
	SuffixCreated                     // counter, histogram, summary
	SuffixCount                       // histogram, summary
	SuffixSum                         // histogram, summary
	SuffixBucket                      // histogram, gaugehistogram
	_SuffixGCount                     // gaugehistogram
	_SuffixGSum                       // gaugehistogram
	SuffixInfo                        // info
	suffixTerminator
)

// A Metric collects metric data.
type Metric interface {
	AppendPoints([]MetricPoint, *Desc) ([]MetricPoint, error)
}

// ----------------------------------------------------------------------------

// A MetricFamily wraps a family of Metrics, where every Metric
// MUST have a unique LabelSet.
type MetricFamily interface {
	// ID returns the numeric metric family ID.
	ID() uint64
	// Desc exposed the metric family description.
	Desc() *Desc
	// Type returns the metric type.
	Type() MetricType
	// NumMetrics returns the number of metrics in the family.
	NumMetrics() int
}

type metricWithLabels struct {
	met Metric
	lvs []string
}

type metricFamily struct {
	desc    Desc
	mt      MetricType
	metrics map[uint64]metricWithLabels
	factory func() (Metric, error)
	onError ErrorHandler

	mu sync.RWMutex
}

func (f *metricFamily) ID() uint64       { return f.desc.calcID() }
func (f *metricFamily) Desc() *Desc      { return &f.desc }
func (f *metricFamily) Type() MetricType { return f.mt }

func (f *metricFamily) NumMetrics() int {
	f.mu.RLock()
	v := len(f.metrics)
	f.mu.RUnlock()
	return v
}

func (f *metricFamily) with(lvs ...string) (Metric, error) {
	labelID := calculateLabelID(len(f.desc.Labels), lvs)

	f.mu.RLock()
	mwl, ok := f.metrics[labelID]
	f.mu.RUnlock()
	if ok {
		return mwl.met, nil
	}

	// with write lock
	f.mu.Lock()
	defer f.mu.Unlock()

	if mwl, ok = f.metrics[labelID]; ok {
		return mwl.met, nil
	}

	err := f.desc.validateLabelValues(lvs)
	if err != nil {
		return nil, err
	}

	met, err := f.factory()
	if err != nil {
		return nil, err
	}

	if f.metrics == nil {
		f.metrics = make(map[uint64]metricWithLabels, 1)
	}
	f.metrics[labelID] = metricWithLabels{met: met, lvs: f.desc.copyLabelValues(lvs)}
	return met, nil
}

func (f *metricFamily) snapshot(s *snapshot) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	s.Reset(f.desc, f.mt)

	if s.cos != nil {
		for id := range f.metrics {
			s.cos = append(s.cos, id)
		}
		sort.Sort(s.cos)

		for _, id := range s.cos {
			m := f.metrics[id]
			if err := s.Append(&m); err != nil {
				return err
			}
		}
	} else {
		for _, m := range f.metrics {
			if err := s.Append(&m); err != nil {
				return err
			}
		}
	}
	return nil
}
