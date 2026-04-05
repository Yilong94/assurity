package application

import (
	"context"
	"errors"
	"testing"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

type fakeLoader struct {
	defs []domain.ServiceDefinition
	err  error
}

func (f *fakeLoader) LoadDefinitions(ctx context.Context) ([]domain.ServiceDefinition, error) {
	return f.defs, f.err
}

type schedRepo struct {
	upsertErr  error
	pending    []int64
	pendingErr error
	updateErr  error
	updated    []int64
}

func (r *schedRepo) Migrate(context.Context) error { return nil }

func (r *schedRepo) UpsertServices(context.Context, []domain.ServiceDefinition) error {
	return r.upsertErr
}

func (r *schedRepo) GetPendingServiceIDs(context.Context) ([]int64, error) {
	return r.pending, r.pendingErr
}

func (r *schedRepo) UpdateServiceEnqueued(_ context.Context, id int64) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updated = append(r.updated, id)
	return nil
}

func (*schedRepo) GetService(context.Context, int64) (domain.ServiceDefinition, error) {
	panic("not used")
}

func (*schedRepo) InsertProbeResult(context.Context, int64, domain.ProbeResult) error {
	panic("not used")
}

func (*schedRepo) GetLatestStatuses(context.Context) ([]domain.ServiceStatus, error) {
	panic("not used")
}

type fakeQueue struct {
	sendErr error
	sent    []domain.ProbeJob
}

func (q *fakeQueue) Send(_ context.Context, job domain.ProbeJob) error {
	if q.sendErr != nil {
		return q.sendErr
	}
	q.sent = append(q.sent, job)
	return nil
}

func (*fakeQueue) Receive(context.Context) (domain.ReceivedProbeJob, error) {
	panic("not used")
}

func (*fakeQueue) Delete(context.Context, string) error { panic("not used") }

func (*fakeQueue) Close() error { return nil }

func TestSchedulerService_Run_loaderError(t *testing.T) {
	s := &SchedulerService{
		Loader: &fakeLoader{err: errors.New("boom")},
		Repo:   &schedRepo{},
		Queue:  &fakeQueue{},
	}
	_, err := s.Run(context.Background())
	if err == nil || err.Error() != "boom" {
		t.Fatalf("err = %v", err)
	}
}

func TestSchedulerService_Run_upsertError(t *testing.T) {
	s := &SchedulerService{
		Loader: &fakeLoader{defs: []domain.ServiceDefinition{{Name: "a", Endpoint: "https://a"}}},
		Repo:   &schedRepo{upsertErr: errors.New("upsert")},
		Queue:  &fakeQueue{},
	}
	_, err := s.Run(context.Background())
	if err == nil || err.Error() != "upsert" {
		t.Fatalf("err = %v", err)
	}
}

func TestSchedulerService_Run_pendingError(t *testing.T) {
	s := &SchedulerService{
		Loader: &fakeLoader{defs: []domain.ServiceDefinition{{Name: "a", Endpoint: "https://a"}}},
		Repo:   &schedRepo{pendingErr: errors.New("query")},
		Queue:  &fakeQueue{},
	}
	_, err := s.Run(context.Background())
	if err == nil || err.Error() != "query" {
		t.Fatalf("err = %v", err)
	}
}

func TestSchedulerService_Run_noPending(t *testing.T) {
	repo := &schedRepo{pending: nil}
	q := &fakeQueue{}
	s := &SchedulerService{
		Loader: &fakeLoader{defs: []domain.ServiceDefinition{{Name: "a", Endpoint: "https://a"}}},
		Repo:   repo,
		Queue:  q,
	}
	n, err := s.Run(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if len(q.sent) != 0 || len(repo.updated) != 0 {
		t.Fatalf("sent=%v updated=%v", q.sent, repo.updated)
	}
}

func TestSchedulerService_Run_enqueuesAndUpdates(t *testing.T) {
	repo := &schedRepo{pending: []int64{10, 20}}
	q := &fakeQueue{}
	s := &SchedulerService{
		Loader: &fakeLoader{defs: []domain.ServiceDefinition{{Name: "a", Endpoint: "https://a"}}},
		Repo:   repo,
		Queue:  q,
	}
	n, err := s.Run(context.Background())
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if len(q.sent) != 2 || q.sent[0].ServiceID != 10 || q.sent[1].ServiceID != 20 {
		t.Fatalf("sent=%v", q.sent)
	}
	if len(repo.updated) != 2 || repo.updated[0] != 10 || repo.updated[1] != 20 {
		t.Fatalf("updated=%v", repo.updated)
	}
}

func TestSchedulerService_Run_sendFails(t *testing.T) {
	repo := &schedRepo{pending: []int64{1, 2}}
	q := &fakeQueue{sendErr: errors.New("sqs")}
	s := &SchedulerService{
		Loader: &fakeLoader{defs: []domain.ServiceDefinition{{Name: "a", Endpoint: "https://a"}}},
		Repo:   repo,
		Queue:  q,
	}
	n, err := s.Run(context.Background())
	if err == nil || err.Error() != "sqs" || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if len(q.sent) != 0 {
		t.Fatalf("sent=%v", q.sent)
	}
}

func TestSchedulerService_Run_updateFails(t *testing.T) {
	repo := &schedRepo{pending: []int64{7}, updateErr: errors.New("db")}
	q := &fakeQueue{}
	s := &SchedulerService{
		Loader: &fakeLoader{defs: []domain.ServiceDefinition{{Name: "a", Endpoint: "https://a"}}},
		Repo:   repo,
		Queue:  q,
	}
	n, err := s.Run(context.Background())
	if err == nil || err.Error() != "db" || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if len(q.sent) != 1 || q.sent[0].ServiceID != 7 {
		t.Fatalf("sent=%v", q.sent)
	}
}

var (
	_ ports.ServiceLoader     = (*fakeLoader)(nil)
	_ ports.ServiceRepository = (*schedRepo)(nil)
	_ ports.JobQueue          = (*fakeQueue)(nil)
)
