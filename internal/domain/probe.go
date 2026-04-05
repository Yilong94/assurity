package domain

// AvailabilityStatus is the outcome of probing an endpoint.
type AvailabilityStatus string

const (
	StatusUp   AvailabilityStatus = "up"
	StatusDown AvailabilityStatus = "down"
)

// ProbeResult is the result of one logical availability check (possibly after retries).
type ProbeResult struct {
	Status    AvailabilityStatus
	LatencyMs int64
	Err       *string
}
