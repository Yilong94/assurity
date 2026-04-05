package application

import (
	"context"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

// StatusService exposes read models for the status API and dashboard.
type StatusService struct {
	Repo ports.ServiceRepository
}

var _ ports.LatestStatusReader = (*StatusService)(nil)

// GetLatestServiceStatuses returns the latest ping row per service.
func (s *StatusService) GetLatestServiceStatuses(ctx context.Context) ([]domain.ServiceStatus, error) {
	return s.Repo.GetLatestStatuses(ctx)
}
