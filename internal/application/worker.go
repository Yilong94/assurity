package application

import (
	"context"
	"errors"
	"log"
	"time"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

// WorkerService runs availability checks and persists outcomes (queue I/O stays in the driver / cmd layer).
type WorkerService struct {
	Repo  ports.ServiceRepository
	Probe ports.AvailabilityProbe
	Alert ports.DownNotifier
}

// ProcessDelivery runs one check and persists the outcome. Returns nil if the service no longer exists (message should be acked).
func (w *WorkerService) Process(ctx context.Context, msg domain.ReceivedProbeJob) error {
	service, err := w.Repo.GetService(ctx, msg.Job.ServiceID)
	if err != nil {
		if errors.Is(err, domain.ErrServiceNotFound) {
			return nil
		}
		return err
	}

	timeout := time.Duration(service.TimeoutSeconds) * time.Second

	outcome := w.Probe.Run(ctx, service.Endpoint, timeout, service.ExtraRetryAttempts)

	if err := w.Repo.InsertProbeResult(ctx, msg.Job.ServiceID, outcome); err != nil {
		return err
	}

	if outcome.Status == domain.StatusDown && w.Alert != nil {
		payload := domain.DownAlertPayload{
			ServiceID: msg.Job.ServiceID,
			Name:      service.Name,
			Endpoint:  service.Endpoint,
			LatencyMs: outcome.LatencyMs,
			Err:       outcome.Err,
		}
		if err := w.Alert.NotifyDown(ctx, payload); err != nil {
			log.Printf("down alert webhook: %v", err)
		}
	}

	return nil
}
