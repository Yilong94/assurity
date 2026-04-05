package ports

import (
	"context"

	"assurity/assignment/internal/domain"
)

// JobQueue is the outbound port for asynchronous check jobs (e.g. SQS).
type JobQueue interface {
	Send(ctx context.Context, job domain.ProbeJob) error
	Receive(ctx context.Context) (domain.ReceivedProbeJob, error)
	Delete(ctx context.Context, receiptHandle string) error
	Close() error
}
