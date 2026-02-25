package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	admindomain "github.com/loangraph/backend/internal/domain/admin"
)

type AdminAuditRepository struct {
	pool *pgxpool.Pool
}

func NewAdminAuditRepository(pool *pgxpool.Pool) *AdminAuditRepository {
	return &AdminAuditRepository{pool: pool}
}

func (r *AdminAuditRepository) Log(ctx context.Context, in admindomain.AuditLogInput) error {
	q := `
INSERT INTO admin_audit_logs (admin_user_id, action, target_type, target_id, payload)
VALUES (NULLIF($1, '')::uuid, $2, $3, $4, $5::jsonb)
`
	_, err := r.pool.Exec(ctx, q, in.AdminUserID, in.Action, in.TargetType, in.TargetID, in.Payload)
	return err
}
