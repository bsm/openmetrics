package openmetrics_test

import (
	"reflect"
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestStateSet(t *testing.T) {
	met := NewStateSet([]string{"foo", "bar"}, StateSetOptions{})
	if exp, got := 2, met.Len(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	if key := "foo"; !met.Contains(key) {
		t.Fatalf("expected to contain %v", key)
	}
	if key := "bar"; !met.Contains(key) {
		t.Fatalf("expected to contain %v", key)
	}
	if key := "baz"; met.Contains(key) {
		t.Fatalf("expected NOT to contain %v", key)
	}

	if key := "foo"; met.IsEnabled(key) {
		t.Fatalf("expected %v to be disabled", key)
	}

	met.Set("foo", true)
	if key := "foo"; !met.IsEnabled(key) {
		t.Fatalf("expected %v to be enabled", key)
	}

	met.Toggle("foo")
	if key := "foo"; met.IsEnabled(key) {
		t.Fatalf("expected %v to be disabled", key)
	}
}

func TestStateSet_AppendPoints(t *testing.T) {
	met := NewStateSet([]string{"foo", "bar"}, StateSetOptions{})
	met.Toggle("foo")

	got, err := met.AppendPoints(nil, &mockDesc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exp := []MetricPoint{
		{Label: Label{Name: "mock", Value: "foo"}, Value: 1},
		{Label: Label{Name: "mock", Value: "bar"}, Value: 0},
	}; !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func BenchmarkStateSet(b *testing.B) {
	met := NewStateSet([]string{"one", "two"}, StateSetOptions{})
	b.Run("Toggle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			met.Toggle("one")
		}
	})
	b.Run("Toggle parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				met.Toggle("one")
			}
		})
	})

	pts := []MetricPoint{}
	b.Run("AppendPoints", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var err error
			if pts, err = met.AppendPoints(pts[:0], &mockDesc); err != nil {
				b.Fatalf("expected no error, got %v", err)
			}
		}
	})
}
