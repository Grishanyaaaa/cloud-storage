package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// Compile-time check
var _ repository.ShareRepository = (*ShareRepositoryPg)(nil)

type ShareRepositoryPg struct {
	pool *pgxpool.Pool
}

func NewShareRepository(pool *pgxpool.Pool) *ShareRepositoryPg {
	return &ShareRepositoryPg{pool: pool}
}

const shareColumns = `id, node_id, owner_id, token_hash, permission, expires_at, revoked_at, created_at`

func (r *ShareRepositoryPg) Create(ctx context.Context, s *entity.Share) error {
	const q = `
INSERT INTO shares (id, node_id, owner_id, token_hash, permission, expires_at, revoked_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	if _, err := r.pool.Exec(ctx, q,
		s.ID().Value(),
		s.NodeID().Value(),
		s.OwnerID().Value(),
		s.TokenHash(),
		string(s.Permission()),
		nullableTime(s.ExpiresAt()),
		nullableTime(s.RevokedAt()),
		s.CreatedAt(),
	); err != nil {
		return fmt.Errorf("insert share: %w", err)
	}
	return nil
}

func (r *ShareRepositoryPg) GetByID(ctx context.Context, ownerID valueobject.UserID, id valueobject.ShareID) (*entity.Share, error) {
	const q = `SELECT ` + shareColumns + ` FROM shares WHERE id = $1 AND owner_id = $2`
	row := r.pool.QueryRow(ctx, q, id.Value(), ownerID.Value())
	return r.scanRow(row)
}

func (r *ShareRepositoryPg) GetActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*entity.Share, error) {
	const q = `SELECT ` + shareColumns + ` FROM shares WHERE token_hash = $1`
	row := r.pool.QueryRow(ctx, q, tokenHash)
	share, err := r.scanRow(row)
	if err != nil {
		return nil, err
	}
	if err := share.AssertActive(now); err != nil {
		return nil, err
	}
	return share, nil
}

func (r *ShareRepositoryPg) ListByNode(ctx context.Context, ownerID valueobject.UserID, nodeID valueobject.NodeID, includeRevoked bool) ([]*entity.Share, error) {
	q := `SELECT ` + shareColumns + ` FROM shares WHERE owner_id = $1 AND node_id = $2`
	if !includeRevoked {
		q += ` AND revoked_at IS NULL`
	}
	q += ` ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, ownerID.Value(), nodeID.Value())
	if err != nil {
		return nil, fmt.Errorf("query shares by node: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.Share, 0)
	for rows.Next() {
		s, err := r.scanRowFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ShareRepositoryPg) RevokeByID(ctx context.Context, ownerID valueobject.UserID, id valueobject.ShareID, now time.Time) error {
	const q = `UPDATE shares SET revoked_at = $3 WHERE id = $1 AND owner_id = $2 AND revoked_at IS NULL`
	ct, err := r.pool.Exec(ctx, q, id.Value(), ownerID.Value(), now)
	if err != nil {
		return fmt.Errorf("revoke share: %w", err)
	}
	if ct.RowsAffected() == 0 {
		// Either not found or already revoked; differentiate.
		const exists = `SELECT 1 FROM shares WHERE id = $1 AND owner_id = $2`
		var x int
		if scanErr := r.pool.QueryRow(ctx, exists, id.Value(), ownerID.Value()).Scan(&x); scanErr != nil {
			if isNoRows(scanErr) {
				return domainerr.ErrShareNotFound
			}
			return fmt.Errorf("verify share existence: %w", scanErr)
		}
		// Already revoked — idempotent success.
		return nil
	}
	return nil
}

func (r *ShareRepositoryPg) RevokeSubtreeTx(ctx context.Context, tx repository.Transaction, root *entity.Node, now time.Time) (int64, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return 0, err
	}
	const q = `
UPDATE shares
SET revoked_at = $1
WHERE revoked_at IS NULL
  AND node_id IN (
    SELECT id FROM nodes WHERE path = $2 OR path LIKE $3
  )`
	ct, err := pgxTx.Exec(ctx, q, now, root.Path().String(), root.Path().LikePrefix())
	if err != nil {
		return 0, fmt.Errorf("revoke subtree shares: %w", err)
	}
	return ct.RowsAffected(), nil
}

func (r *ShareRepositoryPg) ExpireDue(ctx context.Context, now time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	const q = `
UPDATE shares
SET revoked_at = $1
WHERE id IN (
    SELECT id FROM shares
    WHERE revoked_at IS NULL AND expires_at IS NOT NULL AND expires_at < $1
    LIMIT $2
)`
	ct, err := r.pool.Exec(ctx, q, now, limit)
	if err != nil {
		return 0, fmt.Errorf("expire shares: %w", err)
	}
	return ct.RowsAffected(), nil
}

func (r *ShareRepositoryPg) scanRow(row pgx.Row) (*entity.Share, error) {
	var (
		id, nodeID, ownerID uuid.UUID
		tokenHash           string
		perm                string
		expiresAt           *time.Time
		revokedAt           *time.Time
		createdAt           time.Time
	)
	if err := row.Scan(&id, &nodeID, &ownerID, &tokenHash, &perm, &expiresAt, &revokedAt, &createdAt); err != nil {
		if isNoRows(err) {
			return nil, domainerr.ErrShareNotFound
		}
		return nil, fmt.Errorf("scan share: %w", err)
	}
	p, err := valueobject.ParsePermission(perm)
	if err != nil {
		return nil, err
	}
	return entity.ReconstructShare(
		valueobject.ShareIDFromUUID(id),
		valueobject.NodeIDFromUUID(nodeID),
		valueobject.UserIDFromUUID(ownerID),
		tokenHash,
		p,
		expiresAt,
		revokedAt,
		createdAt,
	), nil
}

func (r *ShareRepositoryPg) scanRowFromRows(rows pgx.Rows) (*entity.Share, error) {
	var (
		id, nodeID, ownerID uuid.UUID
		tokenHash           string
		perm                string
		expiresAt           *time.Time
		revokedAt           *time.Time
		createdAt           time.Time
	)
	if err := rows.Scan(&id, &nodeID, &ownerID, &tokenHash, &perm, &expiresAt, &revokedAt, &createdAt); err != nil {
		return nil, fmt.Errorf("scan share row: %w", err)
	}
	p, err := valueobject.ParsePermission(perm)
	if err != nil {
		return nil, err
	}
	return entity.ReconstructShare(
		valueobject.ShareIDFromUUID(id),
		valueobject.NodeIDFromUUID(nodeID),
		valueobject.UserIDFromUUID(ownerID),
		tokenHash,
		p,
		expiresAt,
		revokedAt,
		createdAt,
	), nil
}
