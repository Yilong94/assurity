package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithCORS_OPTIONS(t *testing.T) {
	var innerCalls int
	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalls++
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/any", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if innerCalls != 0 {
		t.Fatalf("inner handler called %d times", innerCalls)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS origin header")
	}
	if rec.Header().Get("Access-Control-Allow-Methods") != "GET, OPTIONS" {
		t.Fatal("missing CORS methods header")
	}
}

func TestWithCORS_passesThrough(t *testing.T) {
	var innerCalls int
	h := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalls++
		_, _ = w.Write([]byte("inner"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if innerCalls != 1 {
		t.Fatalf("inner calls = %d", innerCalls)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if rec.Body.String() != "inner" {
		t.Fatalf("body = %q", rec.Body.String())
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header on proxied response")
	}
}
