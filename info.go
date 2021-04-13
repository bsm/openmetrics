package openmetrics

// InfoFamily is a metric family of Infos.
type InfoFamily interface {
	MetricFamily

	// With returns a Info for the given label values.
	With(labelValues ...string) (Info, error)
	// Must behaves like With but panics on errors.
	Must(labelValues ...string) Info
}

type infoFamily struct {
	metricFamily
}

func (f *infoFamily) Must(labelValues ...string) Info {
	ist, err := f.With(labelValues...)
	if err != nil {
		panic(err)
	}
	return ist
}

func (f *infoFamily) With(labelValues ...string) (Info, error) {
	ist, err := f.with(labelValues...)
	if err != nil {
		return nil, err
	}
	return ist.(Info), nil
}

// ----------------------------------------------------------------------------

// Info is an Instrument.
type Info interface {
	Instrument
}

type info struct{}

// NewInfo inits a new Info from labels.
func NewInfo() Info {
	return info{}
}

func (t info) AppendPoints(dst []MetricPoint, _ *Desc) ([]MetricPoint, error) {
	return append(dst,
		MetricPoint{Suffix: SuffixInfo, Value: 1},
	), nil
}
