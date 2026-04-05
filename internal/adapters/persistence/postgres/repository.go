package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

// Repository implements ports.ServiceRepository.
type Repository struct {
	db *sql.DB
}

var _ ports.ServiceRepository = (*Repository)(nil)

// NewRepository returns a PostgreSQL-backed repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Migrate creates tables and indexes if they do not exist.
func (r *Repository) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS services (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			endpoint TEXT NOT NULL,
			check_interval_seconds INTEGER NOT NULL DEFAULT 30,
			check_timeout_seconds INTEGER NOT NULL DEFAULT 15,
			retry_attempts INTEGER NOT NULL DEFAULT 0,
			last_enqueued_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS probe_results (
			id BIGSERIAL PRIMARY KEY,
			service_id BIGINT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
			status TEXT NOT NULL CHECK (status IN ('up','down')),
			latency_ms BIGINT,
			error_message TEXT,
			checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_probe_results_service_checked
		 ON probe_results(service_id, checked_at DESC)`,
	}
	for _, s := range stmts {
		if _, err := r.db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// UpsertServices inserts or updates services from resolved config.
func (r *Repository) UpsertServices(ctx context.Context, defs []domain.ServiceDefinition) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt := `
		INSERT INTO services (name, endpoint, check_interval_seconds, check_timeout_seconds, retry_attempts, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (name) DO UPDATE SET
			endpoint = EXCLUDED.endpoint,
			check_interval_seconds = EXCLUDED.check_interval_seconds,
			check_timeout_seconds = EXCLUDED.check_timeout_seconds,
			retry_attempts = EXCLUDED.retry_attempts,
			updated_at = NOW()`

	for _, s := range defs {
		if _, err := tx.ExecContext(ctx, stmt, s.Name, s.Endpoint, s.IntervalSeconds, s.TimeoutSeconds, s.ExtraRetryAttempts); err != nil {
			return fmt.Errorf("upsert service %q: %w", s.Name, err)
		}
	}
	return tx.Commit()
}

// GetPendingServiceIDs returns services whose check interval has elapsed since last enqueue.
func (r *Repository) GetPendingServiceIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id FROM services
		WHERE last_enqueued_at IS NULL
		   OR last_enqueued_at <= NOW() - (check_interval_seconds * INTERVAL '1 second')
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// UpdateServiceEnqueued sets last_enqueued_at after a job was sent to the queue.
func (r *Repository) UpdateServiceEnqueued(ctx context.Context, serviceID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE services SET last_enqueued_at = NOW() WHERE id = $1`, serviceID)
	return err
}

// GetService returns endpoint and check settings for a worker.
func (r *Repository) GetService(ctx context.Context, serviceID int64) (domain.ServiceDefinition, error) {
	var s domain.ServiceDefinition
	err := r.db.QueryRowContext(ctx, `
		SELECT name, endpoint, check_timeout_seconds, retry_attempts FROM services WHERE id = $1`, serviceID,
	).Scan(&s.Name, &s.Endpoint, &s.TimeoutSeconds, &s.ExtraRetryAttempts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ServiceDefinition{}, domain.ErrServiceNotFound
		}
		return domain.ServiceDefinition{}, err
	}
	return s, nil
}

// InsertProbeResult stores one check outcome.
func (r *Repository) InsertProbeResult(ctx context.Context, serviceID int64, outcome domain.ProbeResult) error {
	status := string(outcome.Status)
	l := outcome.LatencyMs
	latency := &l
	errMsg := outcome.Err

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO probe_results (service_id, status, latency_ms, error_message, checked_at)
		VALUES ($1, $2, $3, $4, NOW())`,
		serviceID, status, latency, errMsg,
	)
	return err
}

// GetLatestStatuses returns the most recent ping per service.
func (r *Repository) GetLatestStatuses(ctx context.Context) ([]domain.ServiceStatus, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.name, s.endpoint,
			pr.status, pr.latency_ms, pr.error_message, pr.checked_at
		FROM services s
		LEFT JOIN LATERAL (
			SELECT status, latency_ms, error_message, checked_at
			FROM probe_results
			WHERE service_id = s.id
			ORDER BY checked_at DESC
			LIMIT 1
		) pr ON true
		ORDER BY s.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ServiceStatus
	for rows.Next() {
		var st domain.ServiceStatus
		var status sql.NullString
		var latency sql.NullInt64
		var errMsg sql.NullString
		var checked sql.NullTime
		if err := rows.Scan(&st.ServiceID, &st.Name, &st.Endpoint, &status, &latency, &errMsg, &checked); err != nil {
			return nil, err
		}
		if status.Valid {
			st.Status = &status.String
		}
		if latency.Valid {
			v := latency.Int64
			st.LatencyMs = &v
		}
		if errMsg.Valid {
			s := errMsg.String
			st.ErrorMessage = &s
		}
		if checked.Valid {
			t := checked.Time
			st.CheckedAt = &t
		}
		out = append(out, st)
	}
	return out, rows.Err()
}
