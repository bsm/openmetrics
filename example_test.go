package openmetrics_test

import (
	"bytes"
	"fmt"
	"os"

	"github.com/bsm/openmetrics"
)

func Example() {
	reg := openmetrics.NewConsistentRegistry(mockNow) // or, openmetrics.DefaultRegistry()
	requestCount := reg.Counter(openmetrics.Desc{
		Name:   "http_request",
		Help:   "A counter example",
		Labels: []string{"status"},
	})
	responseTime := reg.Histogram(openmetrics.Desc{
		Name:   "http_request",
		Unit:   "seconds",
		Help:   "A histogram example",
		Labels: []string{"status"},
	}, []float64{.005, .01, .05, .1, .5, 1, 5, 10})

	requestCount.With("200").Add(1)
	responseTime.With("200").Observe(0.56)

	var buf bytes.Buffer
	if _, err := reg.WriteTo(&buf); err != nil {
		panic(err)
	}
	fmt.Print(buf.String())

	// Output:
	// # TYPE http_request counter
	// # HELP http_request A counter example
	// http_request_total{status="200"} 1
	// http_request_created{status="200"} 1515151515.757576
	// # TYPE http_request_seconds histogram
	// # UNIT http_request_seconds seconds
	// # HELP http_request_seconds A histogram example
	// http_request_seconds_bucket{status="200",le="0.005"} 0
	// http_request_seconds_bucket{status="200",le="0.01"} 0
	// http_request_seconds_bucket{status="200",le="0.05"} 0
	// http_request_seconds_bucket{status="200",le="0.1"} 0
	// http_request_seconds_bucket{status="200",le="0.5"} 0
	// http_request_seconds_bucket{status="200",le="1"} 1
	// http_request_seconds_bucket{status="200",le="5"} 1
	// http_request_seconds_bucket{status="200",le="10"} 1
	// http_request_seconds_bucket{status="200",le="+Inf"} 1
	// http_request_seconds_count{status="200"} 1
	// http_request_seconds_sum{status="200"} 0.56
	// http_request_seconds_created{status="200"} 1515151515.757576
	// # EOF
}

func ExampleExemplar() {
	reg := openmetrics.NewConsistentRegistry(mockNow) // or, openmetrics.DefaultRegistry()
	internalError := reg.Counter(openmetrics.Desc{
		Name: "internal_error",
		Help: "A counter example",
	})

	_, err := os.Stat("/.dockerenv")
	if err != nil {
		// the combined length of label names and values of MUST NOT exceed 128 characters
		errmsg := err.Error()
		if len(errmsg) > 120 {
			errmsg = errmsg[:120]
		}

		internalError.With().AddExemplar(&openmetrics.Exemplar{
			Value:  1,
			Labels: openmetrics.Labels("error", errmsg),
		})
	}
}
