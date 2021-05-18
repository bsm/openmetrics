package openmetrics_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	. "github.com/bsm/openmetrics"
)

func TestCounter(t *testing.T) {
	met := NewCounter(CounterOptions{CreatedAt: mockTime})
	if exp, got := mockTime, met.Created(); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}
}

func TestCounter_Add(t *testing.T) {
	acc := new(errorCollector)
	met := NewCounter(CounterOptions{OnError: acc.OnError})
	met.Add(3.5)
	if exp, got := 3.5, met.Total(); exp != got {
		t.Errorf("expected %v, got %v", exp, got)
	}

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

		for _, x := range examples {
			acc.Reset()
			met.Add(x.V)
			if exp, got := []string{x.M}, acc.Errors(); !reflect.DeepEqual(exp, got) {
				t.Errorf("expected %v, got %v", exp, got)
			}
		}
	})
}

func TestCounter_AddExemplar(t *testing.T) {
	acc := new(errorCollector)
	met := NewCounter(CounterOptions{OnError: acc.OnError})

	t.Run("invalid", func(t *testing.T) {
		if got := met.Exemplar(); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}

		acc.Reset()
		met.AddExemplar(&Exemplar{Value: 1.0, Labels: LabelSet{{"bad key", "hi"}}})
		if exp, got := 1.0, met.Total(); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
		if got := met.Exemplar(); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
		if exp, got := []string{`label name "bad key" is invalid`}, acc.Errors(); !reflect.DeepEqual(exp, got) {
			t.Errorf("expected %v, got %v", exp, got)
		}
	})

	t.Run("no labels", func(t *testing.T) {
		if got := met.Exemplar(); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}

		met.AddExemplar(&Exemplar{Value: 3.5})
		if exp, got := 4.5, met.Total(); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
		if exp, got := (&Exemplar{
			Value:  3.5,
			Labels: nil,
		}), met.Exemplar(); !reflect.DeepEqual(exp, got) {
			t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
		}
	})

	t.Run("with labels", func(t *testing.T) {
		met.AddExemplar(&Exemplar{Value: 2.1, Labels: LabelSet{
			{Name: "one", Value: "hi"},
			{Name: "two", Value: "lo"},
		}})
		if exp, got := 6.6, met.Total(); exp != got {
			t.Fatalf("expected %v, got %v", exp, got)
		}
		if exp, got := (&Exemplar{
			Value: 2.1,
			Labels: LabelSet{
				{Name: "one", Value: "hi"},
				{Name: "two", Value: "lo"},
			},
		}), met.Exemplar(); !reflect.DeepEqual(exp, got) {
			t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
		}
	})
}

func TestCounter_AppendPoints(t *testing.T) {
	met := NewCounter(CounterOptions{CreatedAt: mockTime})
	met.Add(7.5)

	ltr := mockTime.Add(time.Minute)
	xls := LabelSet{{Name: "one", Value: "hi"}, {Name: "two", Value: "lo"}}
	met.AddExemplar(&Exemplar{Value: 1.5, Timestamp: ltr, Labels: xls})

	got, err := met.AppendPoints(nil, &mockDesc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if exp := []MetricPoint{
		{
			Suffix:   SuffixTotal,
			Value:    9.0,
			Exemplar: &Exemplar{Value: 1.5, Timestamp: ltr, Labels: xls},
		},
		{Suffix: SuffixCreated, Value: 1515151515.757575757},
	}; !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func BenchmarkCounter(b *testing.B) {
	met := NewCounter(CounterOptions{})
	b.Run("Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			met.Add(1)
		}
	})
	b.Run("Add parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				met.Add(1)
			}
		})
	})

	exemplar := &Exemplar{Value: 1.0, Labels: LabelSet{{Name: "one", Value: "hi"}}}
	b.Run("AddExemplar", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			met.AddExemplar(exemplar)
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

func BenchmarkCounterFamily(b *testing.B) {
	reg := NewRegistry()
	cnt := reg.Counter(Desc{
		Name:   "foo",
		Labels: []string{"one", "two"},
	})
	lvs := []string{"hi", "lo"}

	b.Run("With", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cnt.With(lvs...)
		}
	})
}
