package httpprobe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"assurity/assignment/internal/domain"
)

func TestProbe_Run_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	p := New()
	res := p.Run(context.Background(), srv.URL, 2*time.Second, 0)
	if res.Status != domain.StatusUp {
		t.Fatalf("status = %q, want up", res.Status)
	}
	if res.LatencyMs < 0 {
		t.Fatalf("latency = %d", res.LatencyMs)
	}
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", *res.Err)
	}
}

func TestProbe_Run_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	p := New()
	res := p.Run(context.Background(), srv.URL, 2*time.Second, 0)
	if res.Status != domain.StatusDown {
		t.Fatalf("status = %q, want down", res.Status)
	}
	if res.Err == nil || *res.Err != "HTTP 503" {
		t.Fatalf("err = %v, want HTTP 503", res.Err)
	}
}

func TestProbe_Run_RetriesThenOK(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	p := New()
	res := p.Run(context.Background(), srv.URL, 2*time.Second, 2)
	if res.Status != domain.StatusUp {
		t.Fatalf("status = %q after retries, want up", res.Status)
	}
	if n != 2 {
		t.Fatalf("attempts = %d, want 2", n)
	}
}

func TestProbe_Run_AlreadyCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := New()
	res := p.Run(ctx, srv.URL, 2*time.Second, 0)
	if res.Status != domain.StatusDown {
		t.Fatalf("status = %q, want down", res.Status)
	}
	if res.Err == nil {
		t.Fatal("expected error when context is canceled")
	}
}
