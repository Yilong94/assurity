package domain

// DownAlertPayload is sent to external alert sinks when a probe reports down.
type DownAlertPayload struct {
	ServiceID int64   `json:"service_id"`
	Name      string  `json:"name"`
	Endpoint  string  `json:"endpoint"`
	LatencyMs int64   `json:"latency_ms"`
	Err       *string `json:"error,omitempty"`
}
