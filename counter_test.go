package openmetrics_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	. "github.com/bsm/openmetrics"
)

func TestCounter(t *testing.T) {
	now := mockTime
	ist := NewCounterAt(now)
	if exp, got := 0.0, ist.Total(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := now, ist.Created(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	if err := ist.Add(7.5); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exp, got := 7.5, ist.Total(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := now, ist.Created(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
}

func TestCounter_AddWithExemplar(t *testing.T) {
	ist := NewCounter()
	if got := ist.Exemplar(); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
	if err := ist.AddWithExemplar(1.5, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exp, got := 1.5, ist.Total(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := (&Exemplar{Value: 1.5}), ist.Exemplar(); !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}

	if err := ist.AddWithExemplar(1.6, LabelSet{
		{Name: "one", Value: "hi"},
		{Name: "two", Value: "lo"},
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exp, got := 3.1, ist.Total(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := (&Exemplar{
		Value: 1.6,
		Labels: LabelSet{
			{Name: "one", Value: "hi"},
			{Name: "two", Value: "lo"},
		},
	}), ist.Exemplar(); !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func TestCounter_AddWithExemplarAt(t *testing.T) {
	now := mockTime
	ist := NewCounter()
	if got := ist.Exemplar(); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
	if err := ist.AddWithExemplarAt(1.5, now, LabelSet{
		{Name: "one", Value: "hi"},
		{Name: "two", Value: "lo"},
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exp, got := 1.5, ist.Total(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := (&Exemplar{
		Value:     1.5,
		Timestamp: now,
		Labels: LabelSet{
			{Name: "one", Value: "hi"},
			{Name: "two", Value: "lo"},
		},
	}), ist.Exemplar(); !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func TestCounter_Add(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		examples := []struct {
			V float64
			M string
		}{
			{-3, "counters must be monotonically non-decreasing"},
			{math.NaN(), "counters cannot accept NaN values"},
			{math.Inf(1), "counters cannot accept infinity values"},
			{math.Inf(-1), "counters must be monotonically non-decreasing"},
		}

		ist := NewCounter()
		for _, x := range examples {
			if exp, err := x.M, ist.Add(x.V); err == nil || err.Error() != exp {
				t.Errorf("expected %v, got %v", exp, err)
			}
		}
	})
}

func TestCounter_AppendPoints(t *testing.T) {
	tt0 := mockTime
	tt1 := tt0.Add(47 * time.Minute)

	ist := NewCounterAt(tt0)
	if err := ist.Add(7.5); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	xl := LabelSet{{Name: "one", Value: "hi"}, {Name: "two", Value: "lo"}}
	if err := ist.AddWithExemplarAt(1.5, tt1, xl); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, err := ist.AppendPoints(nil, &mockDesc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exp := []MetricPoint{
		{
			Suffix:   SuffixTotal,
			Value:    9.0,
			Exemplar: &Exemplar{Value: 1.5, Timestamp: tt1, Labels: xl},
		},
		{Suffix: SuffixCreated, Value: 1515151515.757575757},
	}; !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func BenchmarkCounter(b *testing.B) {
	ist := NewCounter()
	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := ist.Add(1); err != nil {
				b.Fatalf("expected no error, got %v", err)
			}
		}
	})
	b.Run("Add parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if err := ist.Add(1); err != nil {
					b.Fatalf("expected no error, got %v", err)
				}
			}
		})
	})

	labels := LabelSet{{Name: "one", Value: "value"}}
	b.Run("AddWithExemplar", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := ist.AddWithExemplar(1.0, labels); err != nil {
				b.Fatalf("expected no error, got %v", err)
			}
		}
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
