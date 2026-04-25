package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

const uniqueViolation = "23505"

// Compile-time check: UserRepositoryPg implements repository.UserRepository
var _ repository.UserRepository = (*UserRepositoryPg)(nil)

type UserRepositoryPg struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepositoryPg {
	return &UserRepositoryPg{pool: pool}
}

func (r *UserRepositoryPg) Create(ctx context.Context, user *entity.User) error {
	const q = `
		INSERT INTO users (id, email, password_hash, created_at, updated_at, last_login, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, q,
		user.ID().String(),
		user.Email().String(),
		user.PasswordHash(),
		user.CreatedAt(),
		user.UpdatedAt(),
		user.LastLogin(),
		user.IsActive(),
	)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
			return domainerr.ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepositoryPg) GetByID(ctx context.Context, id valueobject.UserID) (*entity.User, error) {
	const q = `
		SELECT id, email, password_hash, created_at, updated_at, last_login, is_active
		FROM users WHERE id = $1`

	return r.scanUser(ctx, q, id.String())
}

func (r *UserRepositoryPg) GetByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error) {
	const q = `
		SELECT id, email, password_hash, created_at, updated_at, last_login, is_active
		FROM users WHERE email = $1`

	return r.scanUser(ctx, q, email.String())
}

func (r *UserRepositoryPg) Update(ctx context.Context, user *entity.User) error {
	const q = `
		UPDATE users
		SET email = $2, password_hash = $3, updated_at = $4, last_login = $5, is_active = $6
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		user.ID().String(),
		user.Email().String(),
		user.PasswordHash(),
		user.UpdatedAt(),
		user.LastLogin(),
		user.IsActive(),
	)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
			return domainerr.ErrUserAlreadyExists
		}
		return fmt.Errorf("update user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domainerr.ErrUserNotFound
	}

	return nil
}

func (r *UserRepositoryPg) ExistsByEmail(ctx context.Context, email valueobject.Email) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	if err := r.pool.QueryRow(ctx, q, email.String()).Scan(&exists); err != nil {
		return false, fmt.Errorf("exists by email: %w", err)
	}

	return exists, nil
}

// scanUser общий хелпер для маппинга строки в доменную сущность
func (r *UserRepositoryPg) scanUser(ctx context.Context, query string, args ...any) (*entity.User, error) {
	var (
		id           string
		email        string
		passwordHash string
		createdAt    time.Time
		updatedAt    time.Time
		lastLogin    *time.Time
		isActive     bool
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&id, &email, &passwordHash,
		&createdAt, &updatedAt, &lastLogin, &isActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}

	// реконструируем доменную сущность из сырых данных БД
	userID, err := valueobject.ParseUserID(id)
	if err != nil {
		return nil, fmt.Errorf("parse user id from db: %w", err)
	}

	userEmail, err := valueobject.NewEmail(email)
	if err != nil {
		return nil, fmt.Errorf("parse email from db: %w", err)
	}

	return entity.ReconstructUser(userID, userEmail, passwordHash, createdAt, updatedAt, lastLogin, isActive), nil
}
