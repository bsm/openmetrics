package omhttp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/bsm/openmetrics"
	"github.com/bsm/openmetrics/omhttp"
)

func ExampleNewHandler() {
	// Create test registry and register instrument.
	reg := openmetrics.NewConsistentRegistry(mockNow)
	httpRequests := reg.MustCounter(openmetrics.Desc{
		Name:   "http_requests",
		Labels: []string{"path", "status"},
	})

	// Create a mock http.ServeMux and mount a metrics route.
	mux := http.NewServeMux()
	mux.Handle("/metrics", omhttp.NewHandler(reg))

	// Record a mock request.
	httpRequests.Must("/home", "200").Add(1)

	// GET /metrics endpoint.
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	fmt.Print(w.Body.String())

	// Output:
	// # TYPE http_requests counter
	// http_requests_total{path="/home",status="200"} 1
	// http_requests_created{path="/home",status="200"} 1515151515.757576
	// # EOF
}
