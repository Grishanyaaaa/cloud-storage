package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// Compile-time check
var _ repository.UserRootRepository = (*UserRootRepositoryPg)(nil)

type UserRootRepositoryPg struct {
	pool *pgxpool.Pool
}

func NewUserRootRepository(pool *pgxpool.Pool) *UserRootRepositoryPg {
	return &UserRootRepositoryPg{pool: pool}
}

func (r *UserRootRepositoryPg) CreateTx(ctx context.Context, tx repository.Transaction, ur *entity.UserRoot) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}
	const q = `INSERT INTO user_roots (user_id, root_id, created_at) VALUES ($1, $2, $3)`
	if _, err := pgxTx.Exec(ctx, q, ur.UserID().Value(), ur.RootID().Value(), ur.CreatedAt()); err != nil {
		if isUniqueViolation(err) {
			return domainerr.ErrUserRootAlreadyExists
		}
		return fmt.Errorf("insert user_root: %w", err)
	}
	return nil
}

func (r *UserRootRepositoryPg) GetByUserID(ctx context.Context, userID valueobject.UserID) (*entity.UserRoot, error) {
	const q = `SELECT user_id, root_id, created_at FROM user_roots WHERE user_id = $1`
	var (
		uid       uuid.UUID
		rid       uuid.UUID
		createdAt time.Time
	)
	if err := r.pool.QueryRow(ctx, q, userID.Value()).Scan(&uid, &rid, &createdAt); err != nil {
		if isNoRows(err) {
			return nil, domainerr.ErrUserRootNotFound
		}
		return nil, fmt.Errorf("scan user_root: %w", err)
	}
	return entity.ReconstructUserRoot(
		valueobject.UserIDFromUUID(uid),
		valueobject.NodeIDFromUUID(rid),
		createdAt,
	), nil
}

func (r *UserRootRepositoryPg) ExistsByUserID(ctx context.Context, userID valueobject.UserID) (bool, error) {
	const q = `SELECT 1 FROM user_roots WHERE user_id = $1`
	var x int
	err := r.pool.QueryRow(ctx, q, userID.Value()).Scan(&x)
	if err != nil {
		if isNoRows(err) {
			return false, nil
		}
		return false, fmt.Errorf("exists user_root: %w", err)
	}
	return true, nil
}
