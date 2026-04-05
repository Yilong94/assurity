package domain

// ProbeJob is queued work to check one monitored service.
type ProbeJob struct {
	ServiceID int64 `json:"service_id"`
}

// ReceivedJobMessage is a dequeued job with an opaque handle for acknowledgment.
type ReceivedProbeJob struct {
	Job           ProbeJob
	ReceiptHandle string
}
