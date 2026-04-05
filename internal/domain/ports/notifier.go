package ports

import (
	"context"

	"assurity/assignment/internal/domain"
)

// DownNotifier sends alerts when a monitored endpoint is reported down (e.g. HTTP webhook).
type DownNotifier interface {
	NotifyDown(ctx context.Context, payload domain.DownAlertPayload) error
}
