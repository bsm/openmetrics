package openmetrics

import (
	"fmt"

	"github.com/bsm/openmetrics/internal/metro"
)

// A LabelSet MUST consist of Labels and MAY be empty. Label names MUST be
// unique within a LabelSet.
type LabelSet []Label

// Labels constructs a LabelSet from name-value pairs.
func Labels(nameValuePairs ...string) LabelSet {
	if len(nameValuePairs)%2 == 1 {
		panic("openmetrics.Labels: odd argument count")
	}

	set := make(LabelSet, 0, len(nameValuePairs)/2)
	for i := 0; i < len(nameValuePairs); i += 2 {
		set = set.Append(nameValuePairs[i], nameValuePairs[i+1])
	}
	return set
}

// Validate validates the label set and returns errors on failures.
func (ls LabelSet) Validate() error {
	for i, l := range ls {
		if !isValidLabelName(l.Name) {
			return fmt.Errorf("label name %q is invalid", l.Name)
		}
		if l.IsZero() {
			continue
		}
		if !isValidLabelValue(l.Value) {
			return fmt.Errorf("label value %q of %q is invalid", l.Value, l.Name)
		}
		if i < len(ls)-2 {
			for _, m := range ls[i+1:] {
				if l.Name == m.Name {
					return fmt.Errorf("labels contain duplicate %q", l.Name)
				}
			}
		}
	}
	return nil
}

// Append appends a label to the set and returns the resulting set.
func (ls LabelSet) Append(name, value string) LabelSet {
	return append(ls, Label{Name: name, Value: value})
}

// AppendTo copies the labels set by appending to target returning the result.
func (ls LabelSet) AppendTo(target LabelSet) LabelSet {
	for _, l := range ls {
		if !l.IsZero() {
			target = append(target, l)
		}
	}
	return target
}

// Label is a name-value pair. These are used in multiple places: identifying
// timeseries, value of INFO metrics, and exemplars in Histograms.
type Label struct {
	Name, Value string
}

// IsZero returns true if the Label is empty.
// Empty label values SHOULD be treated as if the label was not present.
func (l Label) IsZero() bool {
	return l.Value == ""
}

// IsValid validates the Label.
func (l Label) IsValid() bool {
	return isValidLabelName(l.Name) && isValidLabelValue(l.Value)
}

// ----------------------------------------------------------------------------

const term byte = 255

func calculateLabelID(numLabels int, lvs []string) (id uint64) {
	for _, lv := range lvs {
		id = metro.HashString(lv, id)
		id = metro.HashByte(term, id)
	}

	for i := len(lvs); i < numLabels; i++ {
		id = metro.HashString("", id)
		id = metro.HashByte(term, id)
	}
	return
}
