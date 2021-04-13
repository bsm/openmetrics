package omhttp

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"time"
)

type InstrumentFunc func(req *http.Request, status, bytes int, elapsed time.Duration)

// Instrument returns request instrumentation a middleware.
func Instrument(fn InstrumentFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := NewResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			fn(r, ww.Status(), ww.BytesWritten(), time.Since(start))
		})
	}
}

// The work is derived from Chi's middleware, source:
// https://github.com/go-chi/chi/blob/master/middleware/wrap_writer.go

// ResponseWriter is a proxy around an http.ResponseWriter with support for instrumentation.
type ResponseWriter interface {
	http.ResponseWriter
	// Status returns the HTTP status of the request, or 0 if one has not
	// yet been sent.
	Status() int
	// BytesWritten returns the total number of bytes sent to the client.
	BytesWritten() int
}

// NewResponseWriter creates a proxy around a http.ResponseWriter.
func NewResponseWriter(w http.ResponseWriter, protoMajor int) ResponseWriter {
	_, isFlusher := w.(http.Flusher)

	rw := responseWriter{ResponseWriter: w}
	if protoMajor == 2 {
		_, isPusher := w.(http.Pusher)
		if isFlusher && isPusher {
			return &flushResponseWriter{rw}
		}
	} else {
		_, isHijacker := w.(http.Hijacker)
		_, isReaderFrom := w.(io.ReaderFrom)
		if isFlusher && isHijacker && isReaderFrom {
			return &flushHijackReadFromResponseWriter{rw}
		}
		if isFlusher && isHijacker {
			return &flushHijackResponseWriter{rw}
		}
		if isHijacker {
			return &hijackResponseWriter{rw}
		}
	}

	if isFlusher {
		return &flushResponseWriter{rw}
	}
	return &rw
}

type responseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	code        int
	bytes       int
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.code = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(buf []byte) (int, error) {
	rw.maybeWriteHeader()

	n, err := rw.ResponseWriter.Write(buf)
	rw.bytes += n
	return n, err
}

func (rw *responseWriter) maybeWriteHeader() {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
}

func (rw *responseWriter) Status() int       { return rw.code }
func (rw *responseWriter) BytesWritten() int { return rw.bytes }

type flushResponseWriter struct{ responseWriter }

func (w *flushResponseWriter) Flush() {
	w.wroteHeader = true
	w.ResponseWriter.(http.Flusher).Flush()
}

type hijackResponseWriter struct{ responseWriter }

func (w *hijackResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

type flushHijackResponseWriter struct{ responseWriter }

func (w *flushHijackResponseWriter) Flush() {
	w.wroteHeader = true
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *flushHijackResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

type flushHijackReadFromResponseWriter struct{ responseWriter }

func (w *flushHijackReadFromResponseWriter) Flush() {
	w.wroteHeader = true
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *flushHijackReadFromResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (w *flushHijackReadFromResponseWriter) Push(target string, opts *http.PushOptions) error {
	return w.ResponseWriter.(http.Pusher).Push(target, opts)
}

func (w *flushHijackReadFromResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	w.maybeWriteHeader()

	n, err := w.ResponseWriter.(io.ReaderFrom).ReadFrom(r)
	w.bytes += int(n)
	return n, err
}
