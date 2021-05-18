package openmetrics

// InfoFamily is a metric family of Infos.
type InfoFamily interface {
	MetricFamily

	// With returns an Info for the given label values.
	With(labelValues ...string) Info
}

type infoFamily struct {
	metricFamily
}

func (f *infoFamily) With(labelValues ...string) Info {
	met, err := f.with(labelValues...)
	if err != nil {
		f.onError(err)
		return nullInfo{}
	}
	return met.(Info)
}

// ----------------------------------------------------------------------------

// InfoOptions configure Info instances.
type InfoOptions struct{}

// Info is a Metric.
type Info interface {
	Metric
}

type info struct{}

// NewInfo inits a new Info from labels.
func NewInfo(_ InfoOptions) Info {
	return info{}
}

func (m info) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	return append(dst,
		MetricPoint{Suffix: SuffixInfo, Value: 1},
	), nil
}

type nullInfo struct{}

func (nullInfo) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) { return dst, nil }
