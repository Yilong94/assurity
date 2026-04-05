package httpprobe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

const defaultTimeout = 15 * time.Second

// Probe implements ports.AvailabilityProbe using HTTP GET.
type Probe struct{}

var _ ports.AvailabilityProbe = (*Probe)(nil)

// New returns an HTTP availability probe.
func New() *Probe {
	return &Probe{}
}

// Probe implements ports.AvailabilityProbe.
func (p *Probe) Run(ctx context.Context, url string, timeout time.Duration, retries int) domain.ProbeResult {
	res := checkWithRetries(ctx, url, timeout, retries)
	if res.ok {
		return domain.ProbeResult{Status: domain.StatusUp, LatencyMs: res.latencyMs}
	}

	var errStr *string
	if res.err != nil {
		s := res.err.Error()
		errStr = &s
	}
	return domain.ProbeResult{Status: domain.StatusDown, LatencyMs: res.latencyMs, Err: errStr}
}

type checkResult struct {
	ok        bool
	latencyMs int64
	err       error
}

func checkWithRetries(ctx context.Context, url string, timeout time.Duration, retries int) checkResult {
	maxAttempts := 1 + retries
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var last checkResult
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(100*attempt) * time.Millisecond

			select {
			case <-ctx.Done():
				last.err = ctx.Err()
				return last
			case <-time.After(backoff):
			}
		}

		last = check(ctx, url, timeout)
		if last.ok {
			return last
		}
	}
	return last
}

func check(ctx context.Context, url string, timeout time.Duration) checkResult {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return checkResult{ok: false, err: err}
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return checkResult{ok: false, latencyMs: latency, err: err}
	}

	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return checkResult{ok: true, latencyMs: latency}
	}

	return checkResult{
		ok:        false,
		latencyMs: latency,
		err:       fmt.Errorf("HTTP %d", resp.StatusCode),
	}
}
