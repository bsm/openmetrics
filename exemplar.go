package openmetrics

import (
	"fmt"
	"time"
	"unicode/utf8"
)

// Examplar value.
type Exemplar struct {
	Value     float64
	Timestamp time.Time
	Labels    LabelSet
}

// Reset resets the exemplar properties.
func (x *Exemplar) Reset() {
	*x = Exemplar{Labels: x.Labels[:0]}
}

// AddLabel adds a label to the exemplar.
func (x *Exemplar) AddLabel(name, value string) {
	x.Labels = x.Labels.Append(name, value)
}

// Validate validates the exemplar. The combined length of label names and
// values of MUST NOT exceed 128 UTF-8 characters.
func (x *Exemplar) Validate() error {
	if err := x.Labels.Validate(); err != nil {
		return err
	}

	// There is a hard 128 UTF-8 character limit on exemplar length
	numRunes := 0
	for _, l := range x.Labels {
		if !l.IsZero() {
			numRunes += utf8.RuneCountInString(l.Name) + utf8.RuneCountInString(l.Value)
		}
	}
	if numRunes > 128 {
		return fmt.Errorf("the combined length of the label names and values exceeds 128 characters")
	}

	return nil
}

func (x *Exemplar) copyFrom(other *Exemplar) {
	*x = Exemplar{
		Value:     other.Value,
		Timestamp: other.Timestamp,
		Labels:    other.Labels.AppendTo(x.Labels[:0]),
	}
}
