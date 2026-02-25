package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loangraph/backend/internal/jobs"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Enqueue(ctx context.Context, topic string, payload []byte) error {
	q := `INSERT INTO outbox_jobs (topic, payload, status) VALUES ($1, $2::jsonb, 'pending')`
	_, err := r.pool.Exec(ctx, q, topic, payload)
	return err
}

func (r *OutboxRepository) ClaimPending(ctx context.Context, limit int32) ([]jobs.OutboxJob, error) {
	if limit <= 0 {
		limit = 20
	}
	q := `
WITH claimed AS (
  SELECT id
  FROM outbox_jobs
  WHERE status = 'pending' AND available_at <= NOW()
  ORDER BY id
  LIMIT $1
  FOR UPDATE SKIP LOCKED
)
UPDATE outbox_jobs j
SET status = 'processing', attempts = attempts + 1, updated_at = NOW()
FROM claimed
WHERE j.id = claimed.id
RETURNING j.id, j.topic, j.payload::text, j.status, j.attempts, COALESCE(j.last_error, ''), j.available_at
`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]jobs.OutboxJob, 0)
	for rows.Next() {
		var item jobs.OutboxJob
		var payloadText string
		if err := rows.Scan(&item.ID, &item.Topic, &payloadText, &item.Status, &item.Attempts, &item.LastError, &item.AvailableAt); err != nil {
			return nil, err
		}
		item.Payload = []byte(payloadText)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OutboxRepository) MarkDone(ctx context.Context, jobID int64) error {
	q := `UPDATE outbox_jobs SET status = 'done', updated_at = NOW(), last_error = NULL WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, jobID)
	return err
}

func (r *OutboxRepository) MarkRetry(ctx context.Context, jobID int64, nextAvailableAt time.Time, lastError string) error {
	q := `UPDATE outbox_jobs SET status = 'pending', available_at = $2, last_error = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, jobID, nextAvailableAt, lastError)
	return err
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, jobID int64, lastError string) error {
	q := `UPDATE outbox_jobs SET status = 'failed', last_error = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, jobID, lastError)
	return err
}
