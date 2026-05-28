package health

import (
	"encoding/json"
	"net/http"
	"time"
)

func RegisterHealthz(register func(string, http.HandlerFunc), serviceName, version string) {
	register("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": serviceName,
			"version": version,
			"time":    time.Now().Format(time.DateTime),
		})
	})
}
