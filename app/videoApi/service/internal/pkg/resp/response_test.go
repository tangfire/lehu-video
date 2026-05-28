package resp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"lehu-video/pkg/apperror"
)

func TestErrorEncoderUsesAppErrorCodeAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/video/1", nil)

	ErrorEncoder(rec, req, apperror.NotFound("视频不存在"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != int(apperror.CodeNotFound) || body.Message != "视频不存在" {
		t.Fatalf("body = %#v", body)
	}
}

func TestErrorEncoderHidesInternalErrorDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/video/1", nil)

	ErrorEncoder(rec, req, errors.New("sql: connection refused with password"))

	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Message != "系统开小差了，请稍后再试" {
		t.Fatalf("message = %q", body.Message)
	}
}

func TestResponseEncoderIncludesRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-ID", "req-test-1")

	if err := ResponseEncoder(rec, req, map[string]string{"status": "ok"}); err != nil {
		t.Fatalf("ResponseEncoder() error = %v", err)
	}

	var body Response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.RequestID != "req-test-1" {
		t.Fatalf("request_id = %q, want req-test-1", body.RequestID)
	}
}
