package openmetrics_test

import (
	"bytes"
	"fmt"

	"github.com/bsm/openmetrics"
)

func Example() {
	reg := openmetrics.NewConsistentRegistry(mockNow) // or, openmetrics.DefaultRegistry()
	requestCount := reg.MustCounter(openmetrics.Desc{
		Name:   "http_request",
		Help:   "A counter example",
		Labels: []string{"status"},
	})
	responseTime := reg.MustHistogram(openmetrics.Desc{
		Name:   "http_request",
		Unit:   "seconds",
		Help:   "A histogram example",
		Labels: []string{"status"},
	}, .005, .01, .05, .1, .5, 1, 5, 10)

	requestCount.Must("200").MustAdd(1)
	responseTime.Must("200").MustObserve(0.56)

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
	// http_request_seconds_bucket{status="200",le="5"} 0
	// http_request_seconds_bucket{status="200",le="10"} 0
	// http_request_seconds_bucket{status="200",le="+Inf"} 0
	// http_request_seconds_count{status="200"} 1
	// http_request_seconds_sum{status="200"} 0.56
	// http_request_seconds_created{status="200"} 1515151515.757576
	// # EOF
}
