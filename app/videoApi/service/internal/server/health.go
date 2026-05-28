package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type ReadinessChecker interface {
	PingMySQL(context.Context) error
	PingRedis(context.Context) error
}

func registerHealthRoutes(register func(string, http.HandlerFunc), serviceName, version string, checker ReadinessChecker) {
	register("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeHealthJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"service": serviceName,
			"version": version,
			"time":    time.Now().Format(time.DateTime),
		})
	})
	register("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		status := "ok"
		statusCode := http.StatusOK
		checks := map[string]string{
			"mysql": "ok",
			"redis": "ok",
		}
		if checker == nil {
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
			checks["mysql"] = "unavailable"
			checks["redis"] = "unavailable"
		} else {
			if err := checker.PingMySQL(ctx); err != nil {
				status = "degraded"
				statusCode = http.StatusServiceUnavailable
				checks["mysql"] = "error"
			}
			if err := checker.PingRedis(ctx); err != nil {
				status = "degraded"
				statusCode = http.StatusServiceUnavailable
				checks["redis"] = "error"
			}
		}

		writeHealthJSON(w, statusCode, map[string]interface{}{
			"status": status,
			"checks": checks,
			"time":   time.Now().Format(time.DateTime),
		})
	})
}

func writeHealthJSON(w http.ResponseWriter, statusCode int, body map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
