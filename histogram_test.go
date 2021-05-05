package openmetrics_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	. "github.com/bsm/openmetrics"
)

func TestHistogram(t *testing.T) {
	now := mockTime
	ist, err := NewHistogramAt(now, .1, .5, 1, 5, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if exp, got := int64(0), ist.Count(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := 0.0, ist.Sum(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := now, ist.Created(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := 6, ist.NumBuckets(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	ist.MustObserve(0.03)
	ist.MustObserve(1.2)
	if exp, got := int64(2), ist.Count(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := 1.23, ist.Sum(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := now, ist.Created(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	ist.MustObserveWithExemplarAt(0.71, now.Add(time.Second), LabelSet{
		{Name: "one", Value: "hi"},
		{Name: "two", Value: "lo"},
	})
	if exp, got := int64(3), ist.Count(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
	if exp, got := 1.94, ist.Sum(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
}

func TestNewHistogram(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		examples := [][]float64{
			{1, 2, 3},
		}

		for i, bounds := range examples {
			ist, err := NewHistogram(bounds...)
			if err != nil {
				t.Fatalf("[%d ]expected no error, got %v", i, err)
			}
			if exp, got := len(bounds)+1, ist.NumBuckets(); exp != got {
				t.Fatalf("[%d ]expected %v, got %v", i, exp, got)
			}
			if err := ist.Observe(1.2); err != nil {
				t.Fatalf("[%d ]expected no error, got %v", i, err)
			}
			if exp, got := int64(1), ist.Count(); exp != got {
				t.Fatalf("[%d ]expected %v, got %v", i, exp, got)
			}
		}
	})

	t.Run("single bound", func(t *testing.T) {
		examples := [][]float64{
			{},
			{math.Inf(1)},
		}

		for i, bounds := range examples {
			ist, err := NewHistogram(bounds...)
			if err != nil {
				t.Fatalf("[%d] expected no error, got %v", i, err)
			}
			if exp, got := 1, ist.NumBuckets(); exp != got {
				t.Fatalf("[%d] expected %v, got %v", i, exp, got)
			}
			if err := ist.Observe(1.2); err != nil {
				t.Fatalf("[%d] expected no error, got %v", i, err)
			}
			if exp, got := int64(1), ist.Count(); exp != got {
				t.Fatalf("[%d] expected %v, got %v", i, exp, got)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		examples := [][]float64{
			{1, 3, 2},                     // not sorted
			{1, 1, 2},                     // not strictly sorted
			{1, math.NaN(), 2},            // contains NaN
			{1, math.Inf(-1), 2},          // contains -Inf
			{1, math.Inf(1), math.Inf(1)}, // contains +Inf twice
			{1, math.Inf(1), 2},           // contains +Inf in the middle
		}

		for i, bounds := range examples {
			if _, err := NewHistogram(bounds...); err == nil {
				t.Errorf("[%d] expected error, but none occurred", i)
			}
		}
	})
}

func TestHistogram_Observe_invalid(t *testing.T) {
	ist, err := NewHistogram(.1, .5, 1, 5, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	examples := []struct {
		V float64
		M string
	}{
		{-3, "histograms cannot accept negative values"},
		{math.NaN(), "histograms cannot accept NaN values"},
		{math.Inf(1), "histograms cannot accept infinity values"},
		{math.Inf(-1), "histograms cannot accept negative values"},
	}
	for i, x := range examples {
		if exp, err := x.M, ist.Observe(x.V); err == nil || err.Error() != exp {
			t.Errorf("[%d] expected %v, got %v", i, exp, err)
		}
	}
}

func TestHistogram_AppendPoints(t *testing.T) {
	tt0 := mockTime
	tt1 := tt0.Add(time.Minute)

	ist, err := NewHistogramAt(tt0, 1, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := ist.Observe(1.2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	xls := LabelSet{{Name: "one", Value: "hi"}}
	if err := ist.ObserveWithExemplarAt(0.7, tt1, xls); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, err := ist.AppendPoints(nil, &mockDesc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if exp := []MetricPoint{
		{
			Suffix:   SuffixBucket,
			Label:    Label{Name: "le", Value: "1"},
			Value:    1,
			Exemplar: &Exemplar{Value: 0.7, Timestamp: tt1, Labels: xls},
		},
		{
			Suffix: SuffixBucket,
			Label:  Label{Name: "le", Value: "2"},
			Value:  1,
		},
		{
			Suffix: SuffixBucket,
			Label:  Label{Name: "le", Value: "+Inf"},
			Value:  0,
		},
		{Suffix: SuffixCount, Value: 2},
		{Suffix: SuffixSum, Value: 1.9},
		{Suffix: SuffixCreated, Value: 1515151515.757575757},
	}; !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func BenchmarkHistogram(b *testing.B) {
	ist, err := NewHistogram(0.5, 2)
	if err != nil {
		b.Fatalf("expected no error, got %v", err)
	}
	b.Run("Observe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := ist.Observe(1); err != nil {
				b.Fatalf("expected no error, got %v", err)
			}
		}
	})
	b.Run("Observe parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if err := ist.Observe(1); err != nil {
					b.Fatalf("expected no error, got %v", err)
				}
			}
		})
	})

	labels := LabelSet{{Name: "one", Value: "value"}}
	b.Run("ObserveWithExemplar", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := ist.ObserveWithExemplar(1.0, labels); err != nil {
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
