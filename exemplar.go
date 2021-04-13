package openmetrics

import (
	"fmt"
	"sync"
	"time"
	"unicode/utf8"
)

var exemplarPool sync.Pool

// Examplar value.
type Exemplar struct {
	Value     float64
	Timestamp time.Time
	Labels    LabelSet
}

func newExemplar(val float64, tt time.Time, labels LabelSet) (*Exemplar, error) {
	var x Exemplar
	if err := x.reset(val, tt, labels); err != nil {
		return nil, err
	}
	return &x, nil
}

func poolExemplar(val float64, tt time.Time, labels LabelSet) (*Exemplar, error) {
	if v := exemplarPool.Get(); v != nil {
		x := v.(*Exemplar)
		if err := x.reset(val, tt, labels); err != nil {
			x.release()
			return nil, err
		}
		return x, nil
	}
	return newExemplar(val, tt, labels)
}

func (x *Exemplar) reset(val float64, tt time.Time, labels LabelSet) error {
	if err := labels.Validate(); err != nil {
		return err
	}

	// There is a hard 128 UTF-8 character limit on exemplar length
	numRunes := 0
	for _, l := range labels {
		if !l.IsZero() {
			numRunes += utf8.RuneCountInString(l.Name) + utf8.RuneCountInString(l.Value)
		}
	}
	if numRunes > 128 {
		return fmt.Errorf("the combined length of the label names and values exceeds 128 characters")
	}

	*x = Exemplar{Value: val, Timestamp: tt, Labels: labels.AppendTo(x.Labels[:0])}
	return nil
}

func (x *Exemplar) release() {
	exemplarPool.Put(x)
}
