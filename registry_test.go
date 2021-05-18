package openmetrics_test

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	. "github.com/bsm/openmetrics"
)

func TestRegistry(t *testing.T) {
	acc := new(errorCollector)
	reg := NewConsistentRegistry(mockNow)
	reg.OnError = acc.OnError

	// register counter foo - OK
	foo := reg.Counter(Desc{
		Name:   "foo",
		Help:   "Helpful.",
		Labels: []string{"status"},
	})

	// register counter foo (with extra unit) - OK
	reg.Counter(Desc{Name: "foo", Unit: "any"})

	// register counter foo again - ERROR
	if _, err := reg.AddCounter(Desc{
		Name:   "foo",
		Labels: []string{"other"},
	}); err == nil || err.Error() != `metric "foo" is already registered` {
		t.Errorf("expected error, got %v", err)
	}

	// register counter without desc - ERROR
	if _, err := reg.AddCounter(Desc{}); err == nil || err.Error() != `metric name "" is invalid` {
		t.Errorf("expected error, got %v", err)
	}

	// get counter foo
	cnt := foo.With("201")
	cnt.Add(1)
	if exp, got := 1.0, cnt.Total(); exp != got {
		t.Fatalf("expected %v, got %v", nil, got)
	}

	// get null counter
	cnt = foo.With("too", "many")
	cnt.Add(1)
	if exp, got := 0.0, cnt.Total(); exp != got {
		t.Fatalf("expected %v, got %v", nil, got)
	}
	if exp, got := []string{`metric "foo" requires 1 label value(s)`}, acc.Errors(); !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected:\n\t%+v, got:\n\t%+v", exp, got)
	}
}

func TestRegistry_Counter(t *testing.T) {
	reg := NewConsistentRegistry(mockNow)
	foo := reg.Counter(Desc{Name: "foo", Help: "Some text and \n some \" escaping"})
	foo.With().Add(17.1)
	bar := reg.Counter(Desc{Name: "bar", Unit: "hits", Labels: []string{"path"}})
	bar.With("/").Add(2)
	bar.With("/about").Add(1)

	checkOutput(t, reg, `
		# TYPE foo counter
		# HELP foo Some text and \n some \" escaping
		foo_total 17.1
		foo_created 1515151515.757576
		# TYPE bar_hits counter
		# UNIT bar_hits hits
		bar_hits_total{path="/"} 2
		bar_hits_created{path="/"} 1515151515.757576
		bar_hits_total{path="/about"} 1
		bar_hits_created{path="/about"} 1515151515.757576
		# EOF
	`)
}

func TestRegistry_Gauge(t *testing.T) {
	reg := NewConsistentRegistry(mockNow)
	foo := reg.Gauge(Desc{Name: "foo", Labels: []string{"a"}})
	foo.With("b").Set(17.1)
	foo.With("c").Set(18.2)
	bar := reg.Gauge(Desc{Name: "bar"})
	bar.With().Set(-1.5)
	baz := reg.Gauge(Desc{Name: "bar", Unit: "bytes"})
	baz.With().Set(4096)

	checkOutput(t, reg, `
		# TYPE foo gauge
		foo{a="b"} 17.1
		foo{a="c"} 18.2
		# TYPE bar gauge
		bar -1.5
		# TYPE bar_bytes gauge
		# UNIT bar_bytes bytes
		bar_bytes 4096
		# EOF
	`)
}

func TestRegistry_Histogram(t *testing.T) {
	reg := NewConsistentRegistry(mockNow)
	foo := reg.Histogram(Desc{Name: "foo", Labels: []string{"a"}}, []float64{0.01, .1, 1, 10, 100})
	rnd := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		foo.With("b").Observe(math.Exp(rnd.NormFloat64()*3 - 3))
	}
	for i := 0; i < 100; i++ {
		foo.With("").Observe(math.Exp(rnd.NormFloat64()*3 - 3))
	}
	foo.With("b").ObserveExemplar(&Exemplar{Value: 0.054})
	foo.With("b").ObserveExemplar(&Exemplar{
		Value:  0.67,
		Labels: LabelSet{{Name: "trace_id", Value: "KOO5S4vxi0o"}},
	})
	foo.With("b").ObserveExemplar(&Exemplar{
		Value:     9.8,
		Timestamp: mockTime.Truncate(time.Second),
		Labels:    LabelSet{{Name: "trace_id", Value: "oHg5SJYRHA0"}},
	})

	checkOutput(t, reg, `
		# TYPE foo histogram
		foo_bucket{a="b",le="0.01"} 32
		foo_bucket{a="b",le="0.1"} 25 # {} 0.054
		foo_bucket{a="b",le="1"} 32 # {trace_id="KOO5S4vxi0o"} 0.67
		foo_bucket{a="b",le="10"} 10 # {trace_id="oHg5SJYRHA0"} 9.8 1515151515
		foo_bucket{a="b",le="100"} 4
		foo_bucket{a="b",le="+Inf"} 0
		foo_count{a="b"} 103
		foo_sum{a="b"} 240.40971549095673
		foo_created{a="b"} 1515151515.757576
		foo_bucket{le="0.01"} 39
		foo_bucket{le="0.1"} 27
		foo_bucket{le="1"} 21
		foo_bucket{le="10"} 8
		foo_bucket{le="100"} 4
		foo_bucket{le="+Inf"} 1
		foo_count 100
		foo_sum 320.0575197521944
		foo_created 1515151515.757576
		# EOF
	`)
}

func TestRegistry_Info(t *testing.T) {
	reg := NewConsistentRegistry(mockNow)
	foo := reg.Info(Desc{Name: "foo", Labels: []string{"component", "ver", "sha"}})
	foo.With("core", "8.2.7", "8b993e3f62af95b815796f97a98fd3c54a9c7062")
	foo.With("auth", "8.1.9", "c8901732ef9109a7fb5c34387e815bf63f77d3f6")

	checkOutput(t, reg, `
		# TYPE foo info
		foo_info{component="core",ver="8.2.7",sha="8b993e3f62af95b815796f97a98fd3c54a9c7062"} 1
		foo_info{component="auth",ver="8.1.9",sha="c8901732ef9109a7fb5c34387e815bf63f77d3f6"} 1
		# EOF
	`)
}

func TestRegistry_StateSet(t *testing.T) {
	reg := NewConsistentRegistry(mockNow)

	foo := reg.StateSet(Desc{Name: "foo", Labels: []string{"a"}}, []string{"one", "two"})
	foo.With("b").Set("one", true)
	foo.With("c").Set("two", true)
	foo.With("")

	checkOutput(t, reg, `
		# TYPE foo stateset
		foo{a="b",foo="one"} 1
		foo{a="b",foo="two"} 0
		foo{a="c",foo="one"} 0
		foo{a="c",foo="two"} 1
		foo{foo="one"} 0
		foo{foo="two"} 0
		# EOF
	`)
}

func TestRegistry_Unknowns(t *testing.T) {
	reg := NewConsistentRegistry(mockNow)
	foo := reg.Unknown(Desc{Name: "foo"})
	foo.With().Set(17.1)

	checkOutput(t, reg, `
		# TYPE foo unknown
		foo 17.1
		# EOF
	`)
}

func BenchmarkRegistry_WriteTo(b *testing.B) {
	reg := NewRegistry()
	for i := 0; i < 10_000; i++ {
		name := fmt.Sprintf("cnt_%04d", i+1)
		cnt := reg.Counter(Desc{Name: name, Unit: "hits", Labels: []string{"a"}})
		cnt.With("b").Add(float64(i / 10))
		cnt.With("c").Add(float64(i / 100))
	}

	buf := new(bytes.Buffer)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buf.Reset()
		b.StartTimer()

		if n, err := reg.WriteTo(buf); err != nil {
			b.Fatalf("expected no error, got %v", err)
		} else if exp, got := int64(2_097_000), n/1000*1000; exp != got {
			b.Fatalf("expected %v, got %v", exp, got)
		}
	}
}

// ----------------------------------------------------------------------------

func checkOutput(t *testing.T, reg *Registry, exp string) {
	t.Helper()

	var buf bytes.Buffer
	if n, err := reg.WriteTo(&buf); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if exp, got := buf.Len(), int(n); exp != got {
		t.Fatalf("expected %v, got %v", exp, got)
	}

	// remove tabs, norm whitespace
	exp = strings.ReplaceAll(exp, "\t", "")
	exp = strings.TrimSpace(exp) + "\n"

	if got := buf.String(); exp != got {
		t.Fatalf("raw/norm output mismatch:\n--> EXPECTED\n%s--> GOT\n%s", exp, got)
	}
}
