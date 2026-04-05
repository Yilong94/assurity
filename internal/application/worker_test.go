package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"assurity/assignment/internal/adapters/webhook"
	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

type workerRepo struct {
	getSvc     domain.ServiceDefinition
	getErr     error
	inserted   []insertRec
	insertErr  error
}

type insertRec struct {
	id      int64
	outcome domain.ProbeResult
}

func (r *workerRepo) Migrate(context.Context) error { return nil }

func (r *workerRepo) UpsertServices(context.Context, []domain.ServiceDefinition) error {
	panic("not used")
}

func (r *workerRepo) GetPendingServiceIDs(context.Context) ([]int64, error) { panic("not used") }

func (r *workerRepo) UpdateServiceEnqueued(context.Context, int64) error { panic("not used") }

func (r *workerRepo) GetService(context.Context, int64) (domain.ServiceDefinition, error) {
	if r.getErr != nil {
		return domain.ServiceDefinition{}, r.getErr
	}
	return r.getSvc, nil
}

func (r *workerRepo) InsertProbeResult(_ context.Context, serviceID int64, outcome domain.ProbeResult) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	r.inserted = append(r.inserted, insertRec{serviceID, outcome})
	return nil
}

func (*workerRepo) GetLatestStatuses(context.Context) ([]domain.ServiceStatus, error) { panic("not used") }

type probeStub struct {
	result      domain.ProbeResult
	lastURL     string
	lastTimeout time.Duration
	lastRetries int
}

func (p *probeStub) Run(_ context.Context, url string, timeout time.Duration, retries int) domain.ProbeResult {
	p.lastURL = url
	p.lastTimeout = timeout
	p.lastRetries = retries
	return p.result
}

type alertStub struct {
	calls []domain.DownAlertPayload
	err   error
}

func (a *alertStub) NotifyDown(_ context.Context, payload domain.DownAlertPayload) error {
	a.calls = append(a.calls, payload)
	return a.err
}

func TestWorkerService_Process_serviceNotFound(t *testing.T) {
	repo := &workerRepo{getErr: domain.ErrServiceNotFound}
	w := &WorkerService{Repo: repo, Probe: &probeStub{}, Alert: &webhook.Noop{}}
	err := w.Process(context.Background(), domain.ReceivedProbeJob{Job: domain.ProbeJob{ServiceID: 99}})
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestWorkerService_Process_getError(t *testing.T) {
	repo := &workerRepo{getErr: errors.New("db")}
	w := &WorkerService{Repo: repo, Probe: &probeStub{}, Alert: &webhook.Noop{}}
	err := w.Process(context.Background(), domain.ReceivedProbeJob{Job: domain.ProbeJob{ServiceID: 1}})
	if err == nil || err.Error() != "db" {
		t.Fatalf("err = %v", err)
	}
}

func TestWorkerService_Process_success(t *testing.T) {
	probe := &probeStub{result: domain.ProbeResult{Status: domain.StatusUp, LatencyMs: 5}}
	repo := &workerRepo{
		getSvc: domain.ServiceDefinition{
			Endpoint:           "https://ex",
			TimeoutSeconds:     11,
			ExtraRetryAttempts: 2,
		},
	}
	w := &WorkerService{Repo: repo, Probe: probe, Alert: &webhook.Noop{}}
	err := w.Process(context.Background(), domain.ReceivedProbeJob{Job: domain.ProbeJob{ServiceID: 42}})
	if err != nil {
		t.Fatal(err)
	}
	if probe.lastURL != "https://ex" || probe.lastRetries != 2 || probe.lastTimeout != 11*time.Second {
		t.Fatalf("probe args: url=%q timeout=%v retries=%d", probe.lastURL, probe.lastTimeout, probe.lastRetries)
	}
	if len(repo.inserted) != 1 || repo.inserted[0].id != 42 {
		t.Fatalf("inserted=%v", repo.inserted)
	}
	if repo.inserted[0].outcome.Status != domain.StatusUp || repo.inserted[0].outcome.LatencyMs != 5 {
		t.Fatalf("%+v", repo.inserted[0].outcome)
	}
}

func TestWorkerService_Process_insertError(t *testing.T) {
	repo := &workerRepo{
		getSvc:    domain.ServiceDefinition{Endpoint: "https://x", TimeoutSeconds: 1},
		insertErr: errors.New("insert failed"),
	}
	w := &WorkerService{Repo: repo, Probe: &probeStub{result: domain.ProbeResult{Status: domain.StatusUp}}, Alert: &webhook.Noop{}}
	err := w.Process(context.Background(), domain.ReceivedProbeJob{Job: domain.ProbeJob{ServiceID: 1}})
	if err == nil || err.Error() != "insert failed" {
		t.Fatalf("err = %v", err)
	}
}

func TestWorkerService_Process_downSendsAlert(t *testing.T) {
	errMsg := "timeout"
	probe := &probeStub{result: domain.ProbeResult{Status: domain.StatusDown, LatencyMs: 3, Err: &errMsg}}
	alerts := &alertStub{}
	repo := &workerRepo{
		getSvc: domain.ServiceDefinition{
			Name:               "svc",
			Endpoint:           "https://ex",
			TimeoutSeconds:     5,
			ExtraRetryAttempts: 0,
		},
	}
	w := &WorkerService{Repo: repo, Probe: probe, Alert: alerts}
	err := w.Process(context.Background(), domain.ReceivedProbeJob{Job: domain.ProbeJob{ServiceID: 7}})
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts.calls) != 1 {
		t.Fatalf("alerts = %d", len(alerts.calls))
	}
	p := alerts.calls[0]
	if p.ServiceID != 7 || p.Name != "svc" || p.Endpoint != "https://ex" || p.LatencyMs != 3 || p.Err == nil || *p.Err != "timeout" {
		t.Fatalf("%+v", p)
	}
}

func TestWorkerService_Process_upNoAlert(t *testing.T) {
	probe := &probeStub{result: domain.ProbeResult{Status: domain.StatusUp, LatencyMs: 1}}
	alerts := &alertStub{}
	repo := &workerRepo{getSvc: domain.ServiceDefinition{Name: "a", Endpoint: "https://a", TimeoutSeconds: 1}}
	w := &WorkerService{Repo: repo, Probe: probe, Alert: alerts}
	if err := w.Process(context.Background(), domain.ReceivedProbeJob{Job: domain.ProbeJob{ServiceID: 1}}); err != nil {
		t.Fatal(err)
	}
	if len(alerts.calls) != 0 {
		t.Fatalf("unexpected alerts: %+v", alerts.calls)
	}
}

var (
	_ ports.ServiceRepository = (*workerRepo)(nil)
	_ ports.AvailabilityProbe = (*probeStub)(nil)
	_ ports.DownNotifier      = (*alertStub)(nil)
)
