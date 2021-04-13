package openmetrics

import (
	"time"
	"unicode/utf8"
)

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return (r >= '0' && r <= '9')
}

func isValidMetricName(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		if !(isAlpha(r) || r == '_' || r == ':' || (i > 0 && isDigit(r))) {
			return false
		}
	}

	return true
}

func isValidLabelName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(isAlpha(r) || (i > 0 && r == '_') || (i > 0 && isDigit(r))) {
			return false
		}
	}
	return true
}

func isValidLabelValue(s string) bool {
	return utf8.ValidString(s)
}

func isValidMetricUnit(s string) bool {
	for _, r := range s {
		if !(isAlpha(r) || r == '_' || r == ':' || isDigit(r)) {
			return false
		}
	}
	return true
}

func isValidMetricHelp(s string) bool {
	return utf8.ValidString(s)
}

func asEpoch(t time.Time) float64 {
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9
}

type uint64Slice []uint64

func (s uint64Slice) Len() int           { return len(s) }
func (s uint64Slice) Less(i, j int) bool { return s[i] < s[j] }
func (s uint64Slice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
