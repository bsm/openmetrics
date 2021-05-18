package openmetrics_test

import (
	"math"
	"reflect"
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestGauge(t *testing.T) {
	met := NewGauge(GaugeOptions{})
	if got := met.Value(); !math.IsNaN(got) {
		t.Fatalf("expected NaN, got %v", got)
	}

	met.Set(21)
	if exp, got := 21.0, met.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	met.Set(32)
	if exp, got := 32.0, met.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	met.Add(7.5)
	if exp, got := 39.5, met.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	met.Add(-2.1)
	if exp, got := 37.4, met.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	met.Add(-101.3)
	if exp, got := -63.9, met.Value(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	met.Set(math.NaN())
	if got := met.Value(); !math.IsNaN(got) {
		t.Fatalf("expected NaN, got %v", got)
	}
}

func TestGauge_AppendPoints(t *testing.T) {
	met := NewGauge(GaugeOptions{})
	if got, err := met.AppendPoints(nil, &mockDesc); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if got != nil {
		t.Fatalf("expected %v, got %v", nil, got)
	}

	met.Set(2.4)
	if got, err := met.AppendPoints(nil, &mockDesc); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if exp := []MetricPoint{
		{Value: 2.4},
	}; !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected %+v, got %+v", exp, got)
	}

	met.Set(math.NaN())
	if got, err := met.AppendPoints(nil, &mockDesc); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if len(got) != 1 {
		t.Fatalf("expected %v to have 1 item", got)
	}
}

func BenchmarkGauge(b *testing.B) {
	met := NewGauge(GaugeOptions{})
	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			met.Add(1.0)
		}
	})
	b.Run("Add parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				met.Add(1.0)
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
