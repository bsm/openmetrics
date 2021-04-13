package openmetrics_test

import (
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestDesc_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		examples := []Desc{
			{Name: "foo", Unit: "seconds", Help: "short and useful", Labels: []string{"one", "two"}},
			{Name: "foo123"},
			{Name: "foo_bar"},
			{Name: "foo_123"},
			{Name: "_foo_"},
			{Name: ":foo:"},
		}

		for i, ls := range examples {
			if err := ls.Validate(); err != nil {
				t.Errorf("[%d] expected %v to be valid, but %v", i, ls, err)
			}
		}
	})

	t.Run("bad names", func(t *testing.T) {
		examples := []Desc{
			{Name: "with space"},
			{Name: "with-hyphen"},
			{Name: "1leading_digit"},
			{Name: "ambiguous_suffix_total"},
		}

		for i, ls := range examples {
			if err := ls.Validate(); err == nil {
				t.Errorf("[%d] expected %v to be invalid", i, ls)
			}
		}
	})

	t.Run("bad unit", func(t *testing.T) {
		examples := []Desc{
			{Name: "foo", Unit: "m/s"},
		}

		for i, ls := range examples {
			if err := ls.Validate(); err == nil {
				t.Errorf("[%d] expected %v to be invalid", i, ls)
			}
		}
	})

	t.Run("bad help", func(t *testing.T) {
		examples := []Desc{
			{Name: "foo", Help: "not \xff\xfe\xfd helpful"},
		}

		for i, ls := range examples {
			if err := ls.Validate(); err == nil {
				t.Errorf("[%d] expected %v to be invalid", i, ls)
			}
		}
	})

	t.Run("bad label names", func(t *testing.T) {
		examples := []Desc{
			{Name: "foo", Labels: []string{"_reserved"}},
			{Name: "foo", Labels: []string{"one", "two", "one"}},
		}

		for i, ls := range examples {
			if err := ls.Validate(); err == nil {
				t.Errorf("[%d] expected %v to be invalid", i, ls)
			}
		}
	})
}
