package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func accessLogFilter(logger log.Logger) khttp.FilterFunc {
	helper := log.NewHelper(logger)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v1/campus/") {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			traceCtx, span := otel.Tracer("campus-estation.api.access").Start(
				r.Context(),
				r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()
			r = r.WithContext(traceCtx)

			requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if requestID == "" {
				requestID = fmt.Sprintf("%d", time.Now().UnixNano())
				r.Header.Set("X-Request-ID", requestID)
			}
			w.Header().Set("X-Request-ID", requestID)

			rw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			statusCode := rw.statusCode
			errorText := ""
			if statusCode >= http.StatusBadRequest {
				errorText = http.StatusText(statusCode)
			}
			helper.WithContext(r.Context()).Infow(
				"request_id", requestID,
				"trace_id", tracing.TraceID()(r.Context()),
				"span_id", tracing.SpanID()(r.Context()),
				"user_id", "",
				"ip", requestIP(r),
				"method", r.Method,
				"path", r.URL.Path,
				"status", statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"error", errorText,
			)
		})
	}
}

func requestIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > -1 {
		return host[:idx]
	}
	return host
}
