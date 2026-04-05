package application

import (
	"context"
	"errors"
	"testing"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

type statusRepo struct {
	statuses []domain.ServiceStatus
	err      error
}

func (*statusRepo) Migrate(context.Context) error { panic("not used") }

func (*statusRepo) UpsertServices(context.Context, []domain.ServiceDefinition) error { panic("not used") }

func (*statusRepo) GetPendingServiceIDs(context.Context) ([]int64, error) { panic("not used") }

func (*statusRepo) UpdateServiceEnqueued(context.Context, int64) error { panic("not used") }

func (*statusRepo) GetService(context.Context, int64) (domain.ServiceDefinition, error) {
	panic("not used")
}

func (*statusRepo) InsertProbeResult(context.Context, int64, domain.ProbeResult) error {
	panic("not used")
}

func (r *statusRepo) GetLatestStatuses(context.Context) ([]domain.ServiceStatus, error) {
	return r.statuses, r.err
}

func TestStatusService_GetLatestServiceStatuses(t *testing.T) {
	st := "up"
	repo := &statusRepo{
		statuses: []domain.ServiceStatus{
			{ServiceID: 1, Name: "a", Endpoint: "https://a", Status: &st},
		},
	}
	s := &StatusService{Repo: repo}
	got, err := s.GetLatestServiceStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ServiceID != 1 || *got[0].Status != "up" {
		t.Fatalf("%+v", got)
	}
}

func TestStatusService_GetLatestServiceStatuses_error(t *testing.T) {
	repo := &statusRepo{err: errors.New("db")}
	s := &StatusService{Repo: repo}
	_, err := s.GetLatestServiceStatuses(context.Background())
	if err == nil || err.Error() != "db" {
		t.Fatalf("err = %v", err)
	}
}

var _ ports.ServiceRepository = (*statusRepo)(nil)
