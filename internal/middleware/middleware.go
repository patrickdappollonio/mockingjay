package middleware

import (
	"bufio"
	"net"
	"net/http"

	"github.com/justinas/alice"
)

// Middleware represents a configurable middleware component
type Middleware interface {
	// Name returns the middleware name for logging and identification
	Name() string

	// Handler returns the standard Go middleware handler
	Handler() func(http.Handler) http.Handler
}

// ResponseWriter wraps http.ResponseWriter to capture response metadata
// and implements all the optional interfaces that http.ResponseWriter may support
type ResponseWriter struct {
	http.ResponseWriter
	status      int
	size        int
	wroteHeader bool
}

// NewResponseWriter creates a new ResponseWriter wrapper
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

// Status returns the HTTP status code
func (rw *ResponseWriter) Status() int {
	return rw.status
}

// Size returns the number of bytes written
func (rw *ResponseWriter) Size() int {
	return rw.size
}

// Written returns whether the response header has been written
func (rw *ResponseWriter) Written() bool {
	return rw.wroteHeader
}

// WriteHeader captures the status code and calls the underlying WriteHeader
func (rw *ResponseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the size and calls the underlying Write
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Hijack implements http.Hijacker interface
func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flush implements http.Flusher interface
func (rw *ResponseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// CloseNotify implements http.CloseNotifier interface (deprecated but still supported)
func (rw *ResponseWriter) CloseNotify() <-chan bool {
	if notifier, ok := rw.ResponseWriter.(http.CloseNotifier); ok {
		return notifier.CloseNotify()
	}
	// Return a channel that never sends if the underlying ResponseWriter doesn't support it
	return make(<-chan bool)
}

// Push implements http.Pusher interface for HTTP/2 server push
func (rw *ResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// NewChain creates a new Alice middleware chain with automatic ResponseWriter wrapping
func NewChain(middlewares ...Middleware) alice.Chain {
	chain := alice.New()

	// Add ResponseWriter wrapping as the first middleware
	chain = chain.Append(responseWriterMiddleware)

	// Add all provided middlewares
	for _, mw := range middlewares {
		chain = chain.Append(mw.Handler())
	}

	return chain
}

// responseWriterMiddleware automatically wraps the ResponseWriter for status/size capture
func responseWriterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only wrap if not already wrapped
		if _, ok := w.(*ResponseWriter); !ok {
			w = NewResponseWriter(w)
		}
		next.ServeHTTP(w, r)
	})
}
