package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"assurity/assignment/internal/domain"
)

func TestNotifier_NotifyDown(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %q", ct)
		}
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	n := New(srv.URL)
	payload := domain.DownAlertPayload{
		ServiceID: 9,
		Name:      "n",
		Endpoint:  "https://e",
		LatencyMs: 12,
	}
	if err := n.NotifyDown(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	var decoded domain.DownAlertPayload
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ServiceID != 9 || decoded.Name != "n" || decoded.Endpoint != "https://e" || decoded.LatencyMs != 12 {
		t.Fatalf("%+v", decoded)
	}
}

func TestNotifier_NotifyDown_non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	n := New(srv.URL)
	err := n.NotifyDown(context.Background(), domain.DownAlertPayload{ServiceID: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNoop_NotifyDown(t *testing.T) {
	var n Noop
	if err := n.NotifyDown(context.Background(), domain.DownAlertPayload{}); err != nil {
		t.Fatal(err)
	}
}
