package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeReadinessChecker struct {
	mysqlErr error
	redisErr error
}

func (c fakeReadinessChecker) PingMySQL(context.Context) error { return c.mysqlErr }
func (c fakeReadinessChecker) PingRedis(context.Context) error { return c.redisErr }

func TestHealthzReturnsOK(t *testing.T) {
	mux := http.NewServeMux()
	registerHealthRoutes(func(pattern string, handler http.HandlerFunc) {
		mux.HandleFunc(pattern, handler)
	}, "campus-api", "test", fakeReadinessChecker{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" || body["service"] != "campus-api" || body["version"] != "test" {
		t.Fatalf("body = %#v", body)
	}
}

func TestReadyzReturnsOKWhenDependenciesHealthy(t *testing.T) {
	mux := http.NewServeMux()
	registerHealthRoutes(func(pattern string, handler http.HandlerFunc) {
		mux.HandleFunc(pattern, handler)
	}, "campus-api", "test", fakeReadinessChecker{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" || body.Checks["mysql"] != "ok" || body.Checks["redis"] != "ok" {
		t.Fatalf("body = %#v", body)
	}
}

func TestReadyzReturnsUnavailableWhenDependencyFails(t *testing.T) {
	mux := http.NewServeMux()
	registerHealthRoutes(func(pattern string, handler http.HandlerFunc) {
		mux.HandleFunc(pattern, handler)
	}, "campus-api", "test", fakeReadinessChecker{mysqlErr: errors.New("down")})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "degraded" || body.Checks["mysql"] != "error" || body.Checks["redis"] != "ok" {
		t.Fatalf("body = %#v", body)
	}
}
