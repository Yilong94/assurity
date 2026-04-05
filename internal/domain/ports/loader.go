package ports

import (
	"context"

	"assurity/assignment/internal/domain"
)

// ServiceLoader loads service definitions from external configuration (inbound / driving port for the scheduler).
type ServiceLoader interface {
	LoadDefinitions(ctx context.Context) ([]domain.ServiceDefinition, error)
}
