package webhook

import (
	"context"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

// Noop is a DownNotifier that does nothing (use when no webhook is configured).
type Noop struct{}

var _ ports.DownNotifier = (*Noop)(nil)

// NotifyDown implements ports.DownNotifier.
func (*Noop) NotifyDown(context.Context, domain.DownAlertPayload) error { return nil }
