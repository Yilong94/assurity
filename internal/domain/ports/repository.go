package ports

import (
	"context"

	"assurity/assignment/internal/domain"
)

// ServiceRepository persists monitored services and check results (driven / outbound port).
type ServiceRepository interface {
	Migrate(ctx context.Context) error
	UpsertServices(ctx context.Context, defs []domain.ServiceDefinition) error
	GetPendingServiceIDs(ctx context.Context) ([]int64, error)
	UpdateServiceEnqueued(ctx context.Context, serviceID int64) error
	GetService(ctx context.Context, serviceID int64) (domain.ServiceDefinition, error)
	InsertProbeResult(ctx context.Context, serviceID int64, outcome domain.ProbeResult) error
	GetLatestStatuses(ctx context.Context) ([]domain.ServiceStatus, error)
}
