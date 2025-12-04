// Package middleware provides HTTP middleware for the goUserAPI service.
package middleware

import (
	"net/http"
	"sync/atomic"
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int32
	written    int32
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        0,
	}
}

func (rw *ResponseWriter) WriteHeader(code int) {
	atomic.StoreInt32(&rw.statusCode, int32(code))
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	if n > 0 {
		atomic.AddInt32(&rw.written, int32(n))
	}
	return n, err
}

func (rw *ResponseWriter) StatusCode() int {
	return int(atomic.LoadInt32(&rw.statusCode))
}

func (rw *ResponseWriter) BytesWritten() int {
	return int(atomic.LoadInt32(&rw.written))
}

func (rw *ResponseWriter) HasBody() bool {
	return atomic.LoadInt32(&rw.written) > 0
}
