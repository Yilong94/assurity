package ports

import (
	"context"
	"time"

	"assurity/assignment/internal/domain"
)

// AvailabilityProbe performs HTTP (or other) reachability checks (outbound port).
type AvailabilityProbe interface {
	Run(ctx context.Context, url string, perAttemptTimeout time.Duration, extraRetries int) domain.ProbeResult
}
