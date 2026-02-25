package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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
