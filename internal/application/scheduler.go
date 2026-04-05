package application

import (
	"context"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

// SchedulerService orchestrates loading catalog state, persisting it, and enqueueing due checks.
type SchedulerService struct {
	Loader ports.ServiceLoader
	Repo   ports.ServiceRepository
	Queue  ports.JobQueue
}

// RunOnce loads definitions, upserts them, and enqueues one job per due service.
func (s *SchedulerService) Run(ctx context.Context) (enqueued int, err error) {
	defs, err := s.Loader.LoadDefinitions(ctx)
	if err != nil {
		return 0, err
	}

	if err := s.Repo.UpsertServices(ctx, defs); err != nil {
		return 0, err
	}

	ids, err := s.Repo.GetPendingServiceIDs(ctx)
	if err != nil {
		return 0, err
	}

	for _, id := range ids {
		if err := s.Queue.Send(ctx, domain.ProbeJob{ServiceID: id}); err != nil {
			return enqueued, err
		}
		if err := s.Repo.UpdateServiceEnqueued(ctx, id); err != nil {
			return enqueued, err
		}
		enqueued++
	}

	return enqueued, nil
}
