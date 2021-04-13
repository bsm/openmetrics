package openmetrics

import "time"

func NewExemplar(val float64, now time.Time, labels LabelSet) (*Exemplar, error) {
	return newExemplar(val, now, labels)
}
