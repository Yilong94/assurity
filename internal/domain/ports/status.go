package ports

import (
	"context"

	"assurity/assignment/internal/domain"
)

// LatestStatusReader is the port for querying latest per-service status (e.g. HTTP API / dashboard).
type LatestStatusReader interface {
	GetLatestServiceStatuses(ctx context.Context) ([]domain.ServiceStatus, error)
}
