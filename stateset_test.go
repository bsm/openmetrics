package openmetrics_test

import (
	"reflect"
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestStateSet(t *testing.T) {
	ist := NewStateSet("foo", "bar")
	if exp, got := 2, ist.Len(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	if key := "foo"; !ist.Contains(key) {
		t.Fatalf("expected to contain %v", key)
	}
	if key := "bar"; !ist.Contains(key) {
		t.Fatalf("expected to contain %v", key)
	}
	if key := "baz"; ist.Contains(key) {
		t.Fatalf("expected NOT to contain %v", key)
	}

	if key := "foo"; ist.IsEnabled(key) {
		t.Fatalf("expected %v to be disabled", key)
	}

	ist.MustSet("foo", true)
	if key := "foo"; !ist.IsEnabled(key) {
		t.Fatalf("expected %v to be enabled", key)
	}

	ist.MustToggle("foo")
	if key := "foo"; ist.IsEnabled(key) {
		t.Fatalf("expected %v to be disabled", key)
	}

	t.Run("invalid key", func(t *testing.T) {
		if exp, err := `invalid state "missing"`, ist.Set("missing", true); err.Error() != exp {
			t.Fatalf("expected %v, got %v", exp, err)
		}
	})
}

func TestStateSet_AppendPoints(t *testing.T) {
	ist := NewStateSet("foo", "bar")
	if err := ist.Toggle("foo"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, err := ist.AppendPoints(nil, &mockDesc)
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
	ist := NewStateSet("one", "two")
	b.Run("Toggle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := ist.Toggle("one"); err != nil {
				b.Fatalf("expected no error, got %v", err)
			}
		}
	})
	b.Run("Toggle parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if err := ist.Toggle("one"); err != nil {
					b.Fatalf("expected no error, got %v", err)
				}
			}
		})
	})

	pts := []MetricPoint{}
	b.Run("AppendPoints", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var err error
			if pts, err = ist.AppendPoints(pts[:0], &mockDesc); err != nil {
				b.Fatalf("expected no error, got %v", err)
			}
		}
	})
}
