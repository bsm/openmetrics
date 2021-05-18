package openmetrics_test

import (
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestExemplar_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		examples := []Exemplar{
			{Value: 1.2, Labels: LabelSet{{"one", "val"}, {"two", "val"}}},
		}

		for i, x := range examples {
			if err := x.Validate(); err != nil {
				t.Errorf("[%d] expected no error, got %v", i, err)
			}
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		examples := []Exemplar{
			{Value: 1.2, Labels: LabelSet{{"bad-key", "val"}, {"two", "val"}}},
			{Value: 1.2, Labels: LabelSet{{"two", "val"}, {"one", "val"}, {"two", "val"}}}, // duplicate
			{Value: 1.2, Labels: LabelSet{
				{"one", "123456789.123456789.123456789.123456789.123456789.123456789.1"},
				{"two", "123456789.123456789.123456789.123456789.123456789.123456789.12"}, // too long
			}},
		}

		for i, x := range examples {
			if err := x.Validate(); err == nil {
				t.Errorf("[%d] expected error, got %v", i, err)
			}
		}
	})
}
