package domain

import (
	"errors"
	"time"
)

// ErrServiceNotFound is returned when a service ID does not exist in persistence.
var ErrServiceNotFound = errors.New("service not found")

// ServiceDefinition is validated per-site configuration ready for persistence and scheduling.
type ServiceDefinition struct {
	Name               string
	Endpoint           string
	IntervalSeconds    int
	TimeoutSeconds     int
	ExtraRetryAttempts int
}

// ServiceStatus is the latest known state for dashboard / API projection.
type ServiceStatus struct {
	ServiceID    int64      `json:"service_id"`
	Name         string     `json:"name"`
	Endpoint     string     `json:"endpoint"`
	Status       *string    `json:"status,omitempty"`
	LatencyMs    *int64     `json:"latency_ms,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CheckedAt    *time.Time `json:"checked_at,omitempty"`
}
