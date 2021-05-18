package openmetrics_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	. "github.com/bsm/openmetrics"
)

func TestHistogram(t *testing.T) {
	met, err := NewHistogram([]float64{.1, .5, 1, 5, 10}, HistogramOptions{CreatedAt: mockTime})
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
	if exp, got := 6, met.NumBuckets(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
}

func TestNewHistogram(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		examples := [][]float64{
			{1, 2, 3},
		}

		for i, bounds := range examples {
			met, err := NewHistogram(bounds, HistogramOptions{})
			if err != nil {
				t.Fatalf("[%d ]expected no error, got %v", i, err)
			}
			if exp, got := len(bounds)+1, met.NumBuckets(); exp != got {
				t.Fatalf("[%d ]expected %v, got %v", i, exp, got)
			}
			met.Observe(1.2)
			if exp, got := int64(1), met.Count(); exp != got {
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
			met, err := NewHistogram(bounds, HistogramOptions{})
			if err != nil {
				t.Fatalf("[%d] expected no error, got %v", i, err)
			}
			if exp, got := 1, met.NumBuckets(); exp != got {
				t.Fatalf("[%d] expected %v, got %v", i, exp, got)
			}
			met.Observe(1.2)
			if exp, got := int64(1), met.Count(); exp != got {
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
			if _, err := NewHistogram(bounds, HistogramOptions{}); err == nil {
				t.Errorf("[%d] expected error, but none occurred", i)
			}
		}
	})
}

func TestHistogram_Observe(t *testing.T) {
	acc := new(errorCollector)
	met, err := NewHistogram([]float64{.1, .5, 1, 5, 10}, HistogramOptions{CreatedAt: mockTime, OnError: acc.OnError})
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
			{-3, "histograms cannot accept negative values"},
			{math.NaN(), "histograms cannot accept NaN values"},
			{math.Inf(1), "histograms cannot accept infinity values"},
			{math.Inf(-1), "histograms cannot accept negative values"},
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

func TestHistogram_ObserveExemplar(t *testing.T) {
	acc := new(errorCollector)
	met, err := NewHistogram([]float64{1, 2, 3}, HistogramOptions{OnError: acc.OnError})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Run("invalid", func(t *testing.T) {
		if got := met.Exemplar(1); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}

		acc.Reset()
		met.ObserveExemplar(&Exemplar{Value: 1.5, Labels: LabelSet{{"bad key", "hi"}}})
		if exp, got := 1.5, met.Sum(); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
		if got := met.Exemplar(1); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
		if exp, got := []string{`label name "bad key" is invalid`}, acc.Errors(); !reflect.DeepEqual(exp, got) {
			t.Errorf("expected %v, got %v", exp, got)
		}
	})

	t.Run("no labels", func(t *testing.T) {
		if got := met.Exemplar(1); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}

		met.ObserveExemplar(&Exemplar{Value: 1.4})
		if exp, got := 2.9, met.Sum(); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
		if exp, got := (&Exemplar{
			Value:  1.4,
			Labels: nil,
		}), met.Exemplar(1); !reflect.DeepEqual(exp, got) {
			t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
		}
	})

	t.Run("with labels", func(t *testing.T) {
		met.ObserveExemplar(&Exemplar{Value: 1.7, Labels: LabelSet{
			{Name: "one", Value: "hi"},
			{Name: "two", Value: "lo"},
		}})
		if exp, got := 4.6, met.Sum(); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
		if exp, got := (&Exemplar{
			Value: 1.7,
			Labels: LabelSet{
				{Name: "one", Value: "hi"},
				{Name: "two", Value: "lo"},
			},
		}), met.Exemplar(1); !reflect.DeepEqual(exp, got) {
			t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
		}
	})
}

func TestHistogram_AppendPoints(t *testing.T) {
	met, err := NewHistogram([]float64{1, 2}, HistogramOptions{CreatedAt: mockTime})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	met.Observe(1.2)

	ltr := mockTime.Add(time.Minute)
	xls := LabelSet{{Name: "one", Value: "hi"}}
	met.ObserveExemplar(&Exemplar{Value: 0.7, Timestamp: ltr, Labels: xls})

	got, err := met.AppendPoints(nil, &mockDesc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if exp := []MetricPoint{
		{
			Suffix:   SuffixBucket,
			Label:    Label{Name: "le", Value: "1"},
			Value:    1,
			Exemplar: &Exemplar{Value: 0.7, Timestamp: ltr, Labels: xls},
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
	met, err := NewHistogram([]float64{0.5, 2}, HistogramOptions{})
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

	exemplar := &Exemplar{Value: 1.0, Labels: LabelSet{{Name: "one", Value: "hi"}}}
	b.Run("ObserveExemplar", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			met.ObserveExemplar(exemplar)
		}
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
