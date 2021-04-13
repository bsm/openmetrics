package omhttp_test

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bsm/openmetrics"
	"github.com/bsm/openmetrics/omhttp"
)

func TestNewHandler_limitConcurrency(t *testing.T) {
	reg := openmetrics.NewConsistentRegistry(mockNow)
	ep := omhttp.NewHandler(reg, omhttp.LimitConcurrency(1))

	code := int32(http.StatusOK)
	wg := new(sync.WaitGroup)
	for i := 0; i < 1000; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			w := httptest.NewRecorder()
			ep.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
			atomic.CompareAndSwapInt32(&code, 200, int32(w.Code))
		}()

		if atomic.LoadInt32(&code) != 200 {
			break
		}
	}
	wg.Wait()

	if exp, got := http.StatusServiceUnavailable, int(code); exp != got {
		t.Fatalf("expected: %v, got: %v", exp, got)
	}
}

func TestNewHandler_compression(t *testing.T) {
	reg := openmetrics.NewConsistentRegistry(mockNow)
	cnt := reg.MustCounter(openmetrics.Desc{
		Name:   "http_requests",
		Labels: []string{"path"},
	})
	for i := 0; i < 1000; i++ {
		cnt.Must(fmt.Sprintf("/i%d", i)).Add(1)
	}
	ep := omhttp.NewHandler(reg)

	// plaintext
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/metrics", nil)
	ep.ServeHTTP(w, r)
	if exp, got := 90_000, w.Body.Len(); math.Abs(float64(exp-got)) > 1_000 {
		t.Fatalf("expected: %v, got: %v", exp, got)
	}

	// compressed
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/metrics", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	ep.ServeHTTP(w, r)

	if exp, got := "gzip", w.Header().Get("Content-Encoding"); exp != got {
		t.Fatalf("expected: %v, got: %v", exp, got)
	}
	if exp, got := 6_000, w.Body.Len(); math.Abs(float64(exp-got)) > 1_000 {
		t.Fatalf("expected: %v, got: %v", exp, got)
	}
}
