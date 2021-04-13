package openmetrics_test

import (
	"math"
	"reflect"
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestGauge(t *testing.T) {
	ist := NewGauge()
	if got := ist.Value(); !math.IsNaN(got) {
		t.Fatalf("expected NaN, got %v", got)
	}

	ist.Set(21)
	if exp, got := 21.0, ist.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	ist.Set(32)
	if exp, got := 32.0, ist.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	ist.Add(7.5)
	if exp, got := 39.5, ist.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	ist.Add(-2.1)
	if exp, got := 37.4, ist.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	ist.Add(-101.3)
	if exp, got := -63.9, ist.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	ist.Set(math.NaN())
	if got := ist.Value(); !math.IsNaN(got) {
		t.Fatalf("expected NaN, got %v", got)
	}
}

func TestGauge_AppendPoints(t *testing.T) {
	ist := NewGauge()
	if got, err := ist.AppendPoints(nil, &mockDesc); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if got != nil {
		t.Fatalf("expected %v, got %v", nil, got)
	}

	ist.Set(2.4)
	if got, err := ist.AppendPoints(nil, &mockDesc); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if exp := []MetricPoint{
		{Value: 2.4},
	}; !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected %+v, got %+v", exp, got)
	}

	ist.Set(math.NaN())
	if got, err := ist.AppendPoints(nil, &mockDesc); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if len(got) != 1 {
		t.Fatalf("expected %v to have 1 item", got)
	}
}

func BenchmarkGauge(b *testing.B) {
	ist := NewGauge()
	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ist.Add(1.0)
		}
	})
	b.Run("Add parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ist.Add(1.0)
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
