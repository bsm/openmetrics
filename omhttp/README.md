# OpenMetrics HTTP

The `omhttp` package provides convenience helpers for HTTP servers.

## Examples

To expose metrics on an HTTP server endpoint:

```go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/bsm/openmetrics"
	"github.com/bsm/openmetrics/omhttp"
)

func main() {
	// Create test registry and register instrument.
	reg := openmetrics.NewConsistentRegistry(mockNow)
	httpRequests := reg.MustCounter(openmetrics.Desc{
		Name:	"http_requests",
		Labels:	[]string{"path", "status"},
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

}
```

To instrument HTTP servers:

```go
package main

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

func main() {
	reg := openmetrics.NewConsistentRegistry(mockNow)

	// Register instruments.
	httpRequests := reg.MustCounter(openmetrics.Desc{
		Name:	"http_requests",
		Labels:	[]string{"path", "status"},
	})
	httpRequestBytes := reg.MustCounter(openmetrics.Desc{
		Name:	"http_requests",
		Unit:	"bytes",
		Labels:	[]string{"path", "status"},
	})
	httpRequestTimes := reg.MustHistogram(openmetrics.Desc{
		Name:	"http_requests",
		Unit:	"seconds",
		Labels:	[]string{"path", "status"},
	}, .1, .2, 0.5, 1)

	// Create a mock http.ServeMux with two routes.
	mux := http.NewServeMux()
	mux.HandleFunc("/home", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "Welcome Home!") })
	mux.HandleFunc("/about", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "What's this about?") })

	// Init the instrumentation middleware.
	middleware := omhttp.Instrument(func(req *http.Request, status, bytes int, elapsed time.Duration) {
		statusString := strconv.Itoa(status)

		// replace elapsed with fixed values for consistent test output
		if req.RemoteAddr == "192.0.2.1:1234" {
			elapsed = 187 * time.Millisecond
		}

		httpRequests.Must(req.URL.Path, statusString).Add(1)
		httpRequestBytes.Must(req.URL.Path, statusString).Add(float64(bytes))
		httpRequestTimes.Must(req.URL.Path, statusString).Observe(elapsed.Seconds())
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

}
```
