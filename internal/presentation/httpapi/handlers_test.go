package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

type fakeLister struct {
	statuses []domain.ServiceStatus
	err      error
}

func (f *fakeLister) GetLatestServiceStatuses(context.Context) ([]domain.ServiceStatus, error) {
	return f.statuses, f.err
}

func newTestMux(t *testing.T, lister ports.LatestStatusReader) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	api := &API{Status: lister}
	api.Register(mux)
	return mux
}

func TestAPI_health_GET(t *testing.T) {
	mux := newTestMux(t, &fakeLister{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("body = %q", body)
	}
}

func TestAPI_health_wrongMethod(t *testing.T) {
	mux := newTestMux(t, &fakeLister{})
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestAPI_status_GET_json(t *testing.T) {
	st := "up"
	lister := &fakeLister{
		statuses: []domain.ServiceStatus{
			{ServiceID: 1, Name: "svc", Endpoint: "https://x", Status: &st},
		},
	}
	mux := newTestMux(t, lister)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q", ct)
	}
	var body struct {
		Services []domain.ServiceStatus `json:"services"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Services) != 1 || body.Services[0].Name != "svc" {
		t.Fatalf("%+v", body)
	}
}

func TestAPI_status_repoError(t *testing.T) {
	mux := newTestMux(t, &fakeLister{err: errors.New("db")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal error") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestAPI_status_wrongMethod(t *testing.T) {
	mux := newTestMux(t, &fakeLister{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestAPI_dashboard_root(t *testing.T) {
	mux := newTestMux(t, &fakeLister{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "Service uptime") {
		t.Fatal("expected dashboard HTML shell")
	}
}

func TestAPI_dashboard_notFoundPath(t *testing.T) {
	mux := newTestMux(t, &fakeLister{})
	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestAPI_dashboard_wrongMethod(t *testing.T) {
	mux := newTestMux(t, &fakeLister{})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d", rec.Code)
	}
}

var _ ports.LatestStatusReader = (*fakeLister)(nil)
