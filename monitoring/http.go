package monitoring

import (
	"bufio"
	"net"
	"net/http"
)

type HttpResponseWriter struct {
	Writer http.ResponseWriter
	Code   int
}

func (w *HttpResponseWriter) Flush() {
	if flusher, ok := w.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *HttpResponseWriter) Header() http.Header {
	return w.Writer.Header()
}

func (w *HttpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.Writer.(http.Hijacker).Hijack()
}

func (w *HttpResponseWriter) Write(bytes []byte) (int, error) {
	return w.Writer.Write(bytes)
}

func (w *HttpResponseWriter) WriteHeader(code int) {
	w.Writer.WriteHeader(code)
	w.Code = code
}
