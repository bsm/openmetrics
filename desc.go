package openmetrics

import (
	"fmt"
	"strings"

	"github.com/bsm/openmetrics/internal/metro"
)

const helpMaxLen = 140

var (
	errHelpTooLong = fmt.Errorf("help is too long (maximum %d characters)", helpMaxLen)
)

// Desc contains the metric family description.
type Desc struct {
	// Name of the metric (required).
	Name string
	// Unit specifies MetricFamily units.
	Unit string
	// Help is a string and SHOULD be non-empty. It is used to give a brief
	// description of the MetricFamily for human consumption and SHOULD be short
	// enough to be used as a tooltip.
	Help string
	// Names of the labels that will be used with this metric (optional).
	Labels []string
}

// Validate validates the description.
func (d *Desc) Validate() error {
	// ensure name is ABNF valid
	if !isValidMetricName(d.Name) {
		return fmt.Errorf("metric name %q is invalid", d.Name)
	}

	// ensure name contains no ambiguous suffix
	for sfx := SuffixTotal; sfx < suffixTerminator-1; sfx++ {
		if str := sfx.String(); strings.HasSuffix(d.Name, str) {
			return fmt.Errorf("metric name %q contains a ambiguous suffix %q", d.Name, str)
		}
	}

	// ensure name contains no unit suffix
	if d.Unit != "" && strings.HasSuffix(d.Name, d.Unit) && strings.HasSuffix(d.Name[:len(d.Name)-len(d.Unit)], "_") {
		return fmt.Errorf("metric name %q must not include the unit", d.Name)
	}

	// ensure unit name is valid
	if !isValidMetricUnit(d.Unit) {
		return fmt.Errorf("unit %q is invalid", d.Unit)
	}

	// ensure help is valid
	if len(d.Help) > helpMaxLen {
		return errHelpTooLong
	} else if !isValidMetricHelp(d.Help) {
		return fmt.Errorf("help %q contains invalid characters", d.Help)
	}

	// ensure label names are valid and unique
	for i, name := range d.Labels {
		if !isValidLabelName(name) {
			return fmt.Errorf("label name %q is invalid", name)
		}
		if i < len(d.Labels)-2 {
			for _, name2 := range d.Labels[i+1:] {
				if name == name2 {
					return fmt.Errorf("label names contain duplicate %q", name)
				}
			}
		}
	}

	return nil
}

// FullName returns the full metric family name.
func (d *Desc) FullName() string {
	if d.Unit != "" {
		return d.Name + "_" + d.Unit
	}
	return d.Name
}

func (d *Desc) copyLabelValues(values []string) []string {
	newValues := make([]string, len(d.Labels))
	copy(newValues, values)
	return newValues
}

func (d *Desc) validateLabelValues(values []string) (err error) {
	if need, got := len(d.Labels), len(values); got > need {
		return fmt.Errorf("metric %q requires %d label value(s)", d.Name, need)
	}

	for i, lv := range values {
		if !isValidLabelValue(lv) {
			values[i] = lv
			if err == nil {
				err = fmt.Errorf("invalid label value %q", lv)
			}
		}
	}
	return
}

func (d *Desc) calcID() (id uint64) {
	id = metro.HashString(d.Name, id)
	id = metro.HashByte(term, id)
	id = metro.HashString(d.Unit, id)
	return
}

func (d *Desc) writeTo(bw *bufferedWriter, mtype MetricType) (total int, err error) {
	var n int

	// TYPE
	n, err = bw.WriteIntro("# TYPE ", d.Name, d.Unit, mtype.String(), false)
	total += n
	if err != nil {
		return
	}

	// UNIT
	if d.Unit != "" {
		n, err = bw.WriteIntro("# UNIT ", d.Name, d.Unit, d.Unit, false)
		total += n
		if err != nil {
			return
		}
	}

	// HELP
	if d.Help != "" {
		n, err = bw.WriteIntro("# HELP ", d.Name, d.Unit, d.Help, true)
		total += n
		if err != nil {
			return
		}
	}

	return
}
