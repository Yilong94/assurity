package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

const defaultTimeout = 5 * time.Second

// Notifier POSTs JSON to a webhook URL when a service is down.
type Notifier struct {
	url    string
	client *http.Client
}

var _ ports.DownNotifier = (*Notifier)(nil)

// New returns a notifier for the given URL, or nil if url is empty (caller should use a no-op).
func New(url string) *Notifier {
	if url == "" {
		return nil
	}
	return &Notifier{
		url: url,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NotifyDown implements ports.DownNotifier.
func (n *Notifier) NotifyDown(ctx context.Context, payload domain.DownAlertPayload) error {
	if n == nil || n.url == "" {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: unexpected status %d", resp.StatusCode)
	}
	return nil
}
