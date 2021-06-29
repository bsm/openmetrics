package omhttp_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/bsm/openmetrics"
	"github.com/bsm/openmetrics/omhttp"
)

func ExampleInstrument() {
	reg := openmetrics.NewConsistentRegistry(mockNow)

	// Register metrics.
	httpRequests := reg.Counter(openmetrics.Desc{
		Name:   "http_requests",
		Labels: []string{"path", "status"},
	})
	httpRequestBytes := reg.Counter(openmetrics.Desc{
		Name:   "http_requests",
		Unit:   "bytes",
		Labels: []string{"path", "status"},
	})
	httpRequestTimes := reg.Histogram(openmetrics.Desc{
		Name:   "http_requests",
		Unit:   "seconds",
		Labels: []string{"path", "status"},
	}, []float64{.1, .2, 0.5, 1})

	// Create a mock http.ServeMux with two routes.
	mux := http.NewServeMux()
	mux.HandleFunc("/home", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "Welcome Home!") })
	mux.HandleFunc("/about", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "What's this about?") })

	// Init the instrumentation middleware.
	middleware := omhttp.Instrument(func(req *http.Request, status, bytes int, elapsed time.Duration) {
		statusString := strconv.Itoa(status)

		// replace elapsed with fixed values for consistent test output
		if req.RemoteAddr == "192.0.2.1:1234" {
			elapsed = 187 * time.Millisecond
		}

		httpRequests.With(req.URL.Path, statusString).Add(1)
		httpRequestBytes.With(req.URL.Path, statusString).Add(float64(bytes))
		httpRequestTimes.With(req.URL.Path, statusString).Observe(elapsed.Seconds())
	})

	// Serve two requests.
	handler := middleware(mux)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/home", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/missing", nil))

	// Write out buffer.
	buf := new(bytes.Buffer)
	if _, err := reg.WriteTo(buf); err != nil {
		panic(err)
	}
	fmt.Print(buf.String())

	// Output:
	// # TYPE http_requests counter
	// http_requests_total{path="/missing",status="404"} 1
	// http_requests_created{path="/missing",status="404"} 1515151515.757576
	// http_requests_total{path="/home",status="200"} 1
	// http_requests_created{path="/home",status="200"} 1515151515.757576
	// # TYPE http_requests_bytes counter
	// # UNIT http_requests_bytes bytes
	// http_requests_bytes_total{path="/missing",status="404"} 19
	// http_requests_bytes_created{path="/missing",status="404"} 1515151515.757576
	// http_requests_bytes_total{path="/home",status="200"} 13
	// http_requests_bytes_created{path="/home",status="200"} 1515151515.757576
	// # TYPE http_requests_seconds histogram
	// # UNIT http_requests_seconds seconds
	// http_requests_seconds_bucket{path="/missing",status="404",le="0.1"} 0
	// http_requests_seconds_bucket{path="/missing",status="404",le="0.2"} 1
	// http_requests_seconds_bucket{path="/missing",status="404",le="0.5"} 0
	// http_requests_seconds_bucket{path="/missing",status="404",le="1"} 0
	// http_requests_seconds_bucket{path="/missing",status="404",le="+Inf"} 0
	// http_requests_seconds_count{path="/missing",status="404"} 1
	// http_requests_seconds_sum{path="/missing",status="404"} 0.187
	// http_requests_seconds_created{path="/missing",status="404"} 1515151515.757576
	// http_requests_seconds_bucket{path="/home",status="200",le="0.1"} 0
	// http_requests_seconds_bucket{path="/home",status="200",le="0.2"} 1
	// http_requests_seconds_bucket{path="/home",status="200",le="0.5"} 0
	// http_requests_seconds_bucket{path="/home",status="200",le="1"} 0
	// http_requests_seconds_bucket{path="/home",status="200",le="+Inf"} 0
	// http_requests_seconds_count{path="/home",status="200"} 1
	// http_requests_seconds_sum{path="/home",status="200"} 0.187
	// http_requests_seconds_created{path="/home",status="200"} 1515151515.757576
	// # EOF
}
