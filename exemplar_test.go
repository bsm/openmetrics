package openmetrics_test

import (
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestNewExemplar(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		if _, err := NewExemplar(1.2, mockTime, LabelSet{
			{"one", "val"},
			{"two", "val"},
		}); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		if _, err := NewExemplar(1.2, mockTime, LabelSet{
			{"bad-key", "val"},
			{"two", "val"},
		}); err == nil {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("duplicate label names", func(t *testing.T) {
		if _, err := NewExemplar(1.2, mockTime, LabelSet{
			{"two", "val"},
			{"one", "val"},
			{"two", "val"},
		}); err == nil {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("too long", func(t *testing.T) {
		if _, err := NewExemplar(1.2, mockTime, LabelSet{
			{"one", "123456789.123456789.123456789.123456789.123456789.123456789.1"},
			{"two", "123456789.123456789.123456789.123456789.123456789.123456789.12"},
		}); err == nil {
			t.Errorf("expected error, got %v", err)
		}
	})
}
