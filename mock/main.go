package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"assurity/assignment/internal/domain"
)

// --- Tweak these when testing the worker / probes ---
//
//   - Status — set defaultResponseStatus to http.StatusOK (up) or
//     http.StatusServiceUnavailable (503, counts as down; use to test webhooks).
//   - Delay  — set defaultResponseDelay to 0, 500*time.Millisecond, 2*time.Second, …
const (
	defaultResponseStatus = http.StatusOK
	defaultResponseDelay  = 0 * time.Second
)

func main() {
	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/webhook", webhookHandler)

	log.Printf("Mock server: http://localhost:8081/ — status=%d delay=%s", defaultResponseStatus, defaultResponseDelay)
	log.Println("Webhook sink: POST http://localhost:8081/webhook")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

func defaultHandler(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(defaultResponseDelay)

	w.WriteHeader(defaultResponseStatus)
	switch defaultResponseStatus {
	case http.StatusOK:
		fmt.Fprintln(w, "OK")
	default:
		fmt.Fprintln(w, "error")
	}
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload domain.DownAlertPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("webhook: invalid JSON: %v body=%q", err, string(body))
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	errStr := "<nil>"
	if payload.Err != nil {
		errStr = *payload.Err
	}
	log.Printf("webhook [down alert] service_id=%d name=%q endpoint=%q latency_ms=%d error=%s",
		payload.ServiceID, payload.Name, payload.Endpoint, payload.LatencyMs, errStr)

	w.WriteHeader(http.StatusOK)
}
