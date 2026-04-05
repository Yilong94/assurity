package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"assurity/assignment/internal/domain"
)

func newRepo(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewRepository(db), mock
}

func TestRepository_Migrate(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS services`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS probe_results`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE INDEX IF NOT EXISTS idx_probe_results_service_checked`).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_GetService_found(t *testing.T) {
	repo, mock := newRepo(t)
	rows := sqlmock.NewRows([]string{"name", "endpoint", "check_timeout_seconds", "retry_attempts"}).
		AddRow("svc", "https://ex", 12, 2)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT name, endpoint, check_timeout_seconds, retry_attempts FROM services WHERE id = $1`,
	)).WithArgs(int64(3)).WillReturnRows(rows)

	s, err := repo.GetService(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "svc" || s.Endpoint != "https://ex" || s.TimeoutSeconds != 12 || s.ExtraRetryAttempts != 2 {
		t.Fatalf("got %+v", s)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_GetService_notFound(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT name, endpoint, check_timeout_seconds, retry_attempts FROM services WHERE id = $1`,
	)).WithArgs(int64(99)).WillReturnError(sql.ErrNoRows)

	_, err := repo.GetService(context.Background(), 99)
	if !errors.Is(err, domain.ErrServiceNotFound) {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_UpdateServiceEnqueued(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE services SET last_enqueued_at = NOW() WHERE id = $1`)).
		WithArgs(int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateServiceEnqueued(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_GetPendingServiceIDs(t *testing.T) {
	repo, mock := newRepo(t)
	rows := sqlmock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM services`)).WillReturnRows(rows)

	ids, err := repo.GetPendingServiceIDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids = %v", ids)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_InsertProbeResult(t *testing.T) {
	repo, mock := newRepo(t)
	msg := "bad"
	out := domain.ProbeResult{Status: domain.StatusDown, LatencyMs: 42, Err: &msg}
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO probe_results`)).
		WithArgs(int64(9), "down", int64(42), &msg).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.InsertProbeResult(context.Background(), 9, out); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_UpsertServices(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO services`)).
		WithArgs("a", "https://a", 30, 15, 0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	defs := []domain.ServiceDefinition{
		{Name: "a", Endpoint: "https://a", IntervalSeconds: 30, TimeoutSeconds: 15, ExtraRetryAttempts: 0},
	}
	if err := repo.UpsertServices(context.Background(), defs); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_GetLatestStatuses(t *testing.T) {
	repo, mock := newRepo(t)
	now := time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "name", "endpoint", "status", "latency_ms", "error_message", "checked_at"}).
		AddRow(int64(1), "svc", "https://x", "up", int64(10), nil, now)

	mock.ExpectQuery(`SELECT s.id, s.name, s.endpoint`).
		WillReturnRows(rows)

	got, err := repo.GetLatestStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	st := got[0]
	if st.ServiceID != 1 || st.Name != "svc" || st.Endpoint != "https://x" {
		t.Fatalf("metadata = %+v", st)
	}
	if st.Status == nil || *st.Status != "up" {
		t.Fatalf("status = %v", st.Status)
	}
	if st.LatencyMs == nil || *st.LatencyMs != 10 {
		t.Fatalf("latency = %v", st.LatencyMs)
	}
	if st.ErrorMessage != nil {
		t.Fatalf("err = %v", st.ErrorMessage)
	}
	if st.CheckedAt == nil || !st.CheckedAt.Equal(now) {
		t.Fatalf("checkedAt = %v", st.CheckedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
