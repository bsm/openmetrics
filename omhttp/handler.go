package omhttp

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bsm/openmetrics"
)

// DefaultHandler is a short-cut for NewHandler(openmetrics.DefaultRegistry(), opts...).
func DefaultHandler(opts ...HandlerOption) http.Handler {
	return NewHandler(nil, opts...)
}

type handler struct {
	reg *openmetrics.Registry
}

// NewHandler inits a new handler.
func NewHandler(reg *openmetrics.Registry, opts ...HandlerOption) http.Handler {
	var c handlerConfig
	for _, o := range opts {
		o.update(&c)
	}

	if reg == nil {
		reg = openmetrics.DefaultRegistry()
	}

	var h http.Handler = handler{reg: reg}
	if skip := c.noCompression; !skip {
		h = withCompression(h)
	}
	if n := c.limitConcurrency; n > 0 {
		h = limitConcurrency(h, n)
	}
	return h
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Content-Type", openmetrics.ContentType)

	n, err := h.reg.WriteTo(w)
	if err != nil && n == 0 {
		msg := "An internal error has occurred:\n\n" + err.Error()
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

// ----------------------------------------------------------------------------

type handlerConfig struct {
	noCompression    bool
	limitConcurrency int
}

// A HandlerOption configures the handler.
type HandlerOption interface {
	update(*handlerConfig)
}

type inlineHandlerOption func(*handlerConfig)

func (f inlineHandlerOption) update(c *handlerConfig) { f(c) }

// NoCompression disables default compression of the response
// body based on Accept-Encoding request headers.
func NoCompression() HandlerOption {
	return inlineHandlerOption(func(c *handlerConfig) { c.noCompression = true })
}

// LimitConcurrency limits the number of concurrent requests that
// can be made to the handler. The endpoint will start responding
// with 503 Service Unavailable once this limit is exceeded.
func LimitConcurrency(n int) HandlerOption {
	return inlineHandlerOption(func(c *handlerConfig) { c.limitConcurrency = n })
}

// ----------------------------------------------------------------------------

func limitConcurrency(h http.Handler, n int) http.Handler {
	concurrentRequests := new(int32)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		numRequests := int(atomic.AddInt32(concurrentRequests, 1))
		defer atomic.AddInt32(concurrentRequests, -1)

		if n > 0 && numRequests > n {
			http.Error(w, "Number of concurrent requests was exceeded", http.StatusServiceUnavailable)
			return
		}

		h.ServeHTTP(w, r)
	})
}

const (
	headerVary            = "Vary"
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
)

func withCompression(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add(headerVary, headerAcceptEncoding)

		if accept := r.Header.Get(headerAcceptEncoding); strings.Contains(accept, "gzip") {
			w.Header().Set(headerContentEncoding, "gzip")

			z := newGzipResponseWriter(w)
			defer z.Close()

			h.ServeHTTP(z, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

var gzipResponseWriterPool sync.Pool

type gzipResponseWriter struct {
	http.ResponseWriter
	z *gzip.Writer
}

func newGzipResponseWriter(rw http.ResponseWriter) *gzipResponseWriter {
	if v := gzipResponseWriterPool.Get(); v != nil {
		w := v.(*gzipResponseWriter)
		w.z.Reset(rw)
		return w
	}

	z, _ := gzip.NewWriterLevel(rw, gzip.BestSpeed)
	return &gzipResponseWriter{ResponseWriter: rw, z: z}
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.z.Write(b)
}

func (w *gzipResponseWriter) Close() error {
	err := w.z.Close()
	gzipResponseWriterPool.Put(w)
	return err
}
