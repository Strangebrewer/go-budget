package middleware

import (
	"bytes"
	"net/http"
	"strings"
	"time"

	"github.com/Strangebrewer/go-budget/tracer"
	"github.com/go-chi/chi/v5"
)

type tracingWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
	wrote  bool
}

func (rw *tracingWriter) WriteHeader(status int) {
	if !rw.wrote {
		rw.status = status
		rw.wrote = true
		rw.ResponseWriter.WriteHeader(status)
	}
}

func (rw *tracingWriter) Write(b []byte) (int, error) {
	if !rw.wrote {
		rw.status = http.StatusOK
		rw.wrote = true
	}
	if rw.status >= 400 && rw.body.Len() < 512 {
		rw.body.Write(b)
	}
	return rw.ResponseWriter.Write(b)
}

func Tracing(tc *tracer.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" || tc == nil {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rw := &tracingWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			end := time.Now()

			rctx := chi.RouteContext(r.Context())
			pattern := rctx.RoutePattern()
			id := chi.URLParam(r, "id")
			op := r.Method + " " + pattern
			if id != "" {
				op = strings.Replace(op, "{id}", id, 1)
			}

			if rw.status >= 400 {
				tc.SendErrorSpan(traceID, op, strings.TrimSpace(rw.body.String()), start, end)
			} else {
				tc.SendSpan(traceID, op, start, end)
			}
		})
	}
}
