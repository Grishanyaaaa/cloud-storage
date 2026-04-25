package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// Compile-time check: RefreshTokenRepositoryPg implements repository.RefreshTokenRepository
var _ repository.RefreshTokenRepository = (*RefreshTokenRepositoryPg)(nil)

type RefreshTokenRepositoryPg struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepositoryPg {
	return &RefreshTokenRepositoryPg{pool: pool}
}

func (r *RefreshTokenRepositoryPg) Save(ctx context.Context, token *entity.RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, revoked_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, q,
		token.ID().String(),
		token.UserID().String(),
		token.TokenHash(),
		token.ExpiresAt(),
		token.CreatedAt(),
		token.RevokedAt(),
		nullableString(token.IPAddress()),
		nullableString(token.UserAgent()),
	)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepositoryPg) FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at, ip_address, user_agent
		FROM refresh_tokens
		WHERE token_hash = $1`

	return r.scanToken(ctx, q, tokenHash)
}

func (r *RefreshTokenRepositoryPg) RevokeByID(ctx context.Context, id valueobject.RefreshTokenID) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id.String())
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domainerr.ErrRefreshTokenNotFound
	}

	return nil
}

func (r *RefreshTokenRepositoryPg) RevokeAllByUserID(ctx context.Context, userID valueobject.UserID) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`

	// тут не проверяем RowsAffected — если токенов нет, это не ошибка
	_, err := r.pool.Exec(ctx, q, userID.String())
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepositoryPg) scanToken(ctx context.Context, query string, args ...any) (*entity.RefreshToken, error) {
	var (
		id        string
		userID    string
		tokenHash string
		expiresAt time.Time
		createdAt time.Time
		revokedAt *time.Time
		ipAddress *string
		userAgent *string
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&id, &userID, &tokenHash,
		&expiresAt, &createdAt, &revokedAt,
		&ipAddress, &userAgent,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("scan refresh token: %w", err)
	}

	tokenID, err := valueobject.ParseRefreshTokenID(id)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token id: %w", err)
	}

	uid, err := valueobject.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	return entity.ReconstructRefreshToken(
		tokenID, uid, tokenHash,
		expiresAt, createdAt, revokedAt,
		derefString(ipAddress), derefString(userAgent),
	), nil
}
