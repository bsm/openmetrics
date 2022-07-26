package openmetrics_test

import (
	"math"
	"reflect"
	"testing"

	. "github.com/bsm/openmetrics"
)

func TestSummary(t *testing.T) {
	met, err := NewSummary(SummaryOptions{CreatedAt: mockTime})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if exp, got := int64(0), met.Count(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := 0.0, met.Sum(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := mockTime, met.Created(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
}

func TestSummary_Observe(t *testing.T) {
	acc := new(errorCollector)
	met, err := NewSummary(SummaryOptions{CreatedAt: mockTime, OnError: acc.OnError})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	met.Observe(0.03)
	met.Observe(1.2)
	if exp, got := int64(2), met.Count(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := 1.23, met.Sum(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := mockTime, met.Created(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	t.Run("invalid", func(t *testing.T) {
		examples := []struct {
			V float64
			M string
		}{
			{-3, "summaries cannot accept negative values"},
			{math.NaN(), "summaries cannot accept NaN values"},
			{math.Inf(1), "summaries cannot accept infinity values"},
			{math.Inf(-1), "summaries cannot accept negative values"},
		}

		for i, x := range examples {
			acc.Reset()
			met.Observe(x.V)
			if exp, got := []string{x.M}, acc.Errors(); !reflect.DeepEqual(exp, got) {
				t.Errorf("[%d] expected %v, got %v", i, exp, got)
			}
		}
	})
}

func TestSummary_AppendPoints(t *testing.T) {
	met, err := NewSummary(SummaryOptions{CreatedAt: mockTime})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	met.Observe(1.2)
	met.Observe(0.7)

	got, err := met.AppendPoints(nil, &mockDesc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if exp := []MetricPoint{
		{Suffix: SuffixCount, Value: 2},
		{Suffix: SuffixSum, Value: 1.9},
		{Suffix: SuffixCreated, Value: 1515151515.757575757},
	}; !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func BenchmarkSummary(b *testing.B) {
	met, err := NewSummary(SummaryOptions{})
	if err != nil {
		b.Fatalf("expected no error, got %v", err)
	}
	b.Run("Observe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			met.Observe(1)
		}
	})
	b.Run("Observe parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				met.Observe(1)
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
