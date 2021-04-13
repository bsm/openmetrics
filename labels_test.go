package openmetrics_test

import (
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestLabelSet_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		examples := []LabelSet{
			{{Name: "one", Value: "val"}, {Name: "two", Value: "val"}},
			{{Name: "one", Value: "val"}, {Name: "two"}, {Name: "two", Value: "val"}},
		}

		for i, ls := range examples {
			if err := ls.Validate(); err != nil {
				t.Errorf("[%d] expected %v to be valid, but %v", i, ls, err)
			}
		}
	})

	t.Run("bad names", func(t *testing.T) {
		examples := []LabelSet{
			{{Name: "not valid", Value: "val"}},
			{{Value: "val"}},
		}

		for i, ls := range examples {
			if err := ls.Validate(); err == nil {
				t.Errorf("[%d] expected %v to be invalid", i, ls)
			}
		}
	})

	t.Run("duplicates names", func(t *testing.T) {
		examples := []LabelSet{
			{{Name: "one", Value: "val"}, {Name: "two", Value: "val"}, {Name: "one", Value: "val"}},
		}

		for i, ls := range examples {
			if err := ls.Validate(); err == nil {
				t.Errorf("[%d] expected %v to be invalid", i, ls)
			}
		}
	})
}

func TestLabel_IsValid(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		examples := []string{
			"simple",
			"mIxEdCaSe",
			"MiXeDcAse",
			"with_underscores",
			"with_123_nums",
			"x",
			"x1",
			"x_",
		}

		for _, s := range examples {
			x := Label{Name: s}
			if !x.IsValid() {
				t.Errorf("expected %q to be valid", s)
			}
		}
	})

	t.Run("valid values", func(t *testing.T) {
		examples := []string{
			"",
			"word",
			"日本",
		}

		for _, s := range examples {
			x := Label{Name: "one", Value: s}
			if !x.IsValid() {
				t.Errorf("expected %q to be valid", s)
			}
		}
	})

	t.Run("invalid names", func(t *testing.T) {
		examples := []string{
			"",
			"_reserved",
			"8forbidden",
			"no spaces",
			"no-hyphens",
		}

		for _, s := range examples {
			x := Label{Name: s}
			if x.IsValid() {
				t.Errorf("expected %q to be invalid", s)
			}
		}
	})

	t.Run("invalid values", func(t *testing.T) {
		examples := []string{
			"\xff\xfe\xfd",
		}

		for _, s := range examples {
			x := Label{Name: "one", Value: s}
			if x.IsValid() {
				t.Errorf("expected %q to be invalid", s)
			}
		}
	})
}
