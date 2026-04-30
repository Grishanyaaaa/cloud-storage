package database

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

	ip, err := parseIPToInet(token.IPAddress())
	if err != nil {
		return fmt.Errorf("invalid ip address: %w", err)
	}

	_, err = r.pool.Exec(ctx, q,
		token.ID().String(),
		token.UserID().String(),
		token.TokenHash(),
		token.ExpiresAt(),
		token.CreatedAt(),
		token.RevokedAt(),
		ip,
		nullableString(token.UserAgent()),
	)
	if err != nil {
		// Check for unique violation on token_hash
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
			return fmt.Errorf("duplicate refresh token hash (collision): %w", err)
		}
		return fmt.Errorf("save refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepositoryPg) SaveTx(ctx context.Context, tx repository.Transaction, token *entity.RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, revoked_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	ip, err := parseIPToInet(token.IPAddress())
	if err != nil {
		return fmt.Errorf("invalid ip address: %w", err)
	}

	pgxTx := unwrapTx(tx)
	_, err = pgxTx.Exec(ctx, q,
		token.ID().String(),
		token.UserID().String(),
		token.TokenHash(),
		token.ExpiresAt(),
		token.CreatedAt(),
		token.RevokedAt(),
		ip,
		nullableString(token.UserAgent()),
	)
	if err != nil {
		// Check for unique violation on token_hash
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
			return fmt.Errorf("duplicate refresh token hash (collision): %w", err)
		}
		return fmt.Errorf("save refresh token in transaction: %w", err)
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

func (r *RefreshTokenRepositoryPg) RevokeByID(ctx context.Context, id valueobject.RefreshTokenID, now time.Time) error {
	// Использование COALESCE гарантирует, что если токен уже был отозван, 
	// мы сохраним его оригинальное время отзыва (не перезапишем на now).
	// RETURNING id позволяет понять, существует ли токен вообще, за 1 запрос.
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = COALESCE(revoked_at, $2)
		WHERE id = $1
		RETURNING id`

	var returnedID string
	err := r.pool.QueryRow(ctx, q, id.String(), now).Scan(&returnedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domainerr.ErrRefreshTokenNotFound
		}
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepositoryPg) RevokeAllByUserID(ctx context.Context, userID valueobject.UserID, now time.Time) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL`

	// тут не проверяем RowsAffected — если токенов нет, это не ошибка
	_, err := r.pool.Exec(ctx, q, userID.String(), now)
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepositoryPg) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	const q = `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1`

	tag, err := r.pool.Exec(ctx, q, before)
	if err != nil {
		return 0, fmt.Errorf("delete expired refresh tokens: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *RefreshTokenRepositoryPg) RevokeByHash(ctx context.Context, tokenHash string, now time.Time) (*entity.RefreshToken, *time.Time, error) {
	// Атомарно помечаем токен как отозванный (если он еще не отозван)
	// и возвращаем его состояние ДО обновления (was_revoked_at) и после него.
	const q = `
		WITH old_data AS (
			SELECT id, revoked_at FROM refresh_tokens WHERE token_hash = $1 FOR UPDATE
		),
		updated AS (
			UPDATE refresh_tokens
			SET revoked_at = COALESCE(refresh_tokens.revoked_at, $2)
			FROM old_data
			WHERE refresh_tokens.id = old_data.id
			RETURNING refresh_tokens.id, refresh_tokens.user_id, refresh_tokens.token_hash,
			          refresh_tokens.expires_at, refresh_tokens.created_at, refresh_tokens.revoked_at,
			          refresh_tokens.ip_address, refresh_tokens.user_agent, old_data.revoked_at AS was_revoked_at
		)
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at,
		       ip_address, user_agent, was_revoked_at FROM updated`

	var (
		id           string
		userID       string
		hash         string
		expiresAt    time.Time
		createdAt    time.Time
		revokedAt    *time.Time
		ipAddress     *netip.Prefix
		userAgent    *string
		wasRevokedAt *time.Time
	)

	err := r.pool.QueryRow(ctx, q, tokenHash, now).Scan(
		&id, &userID, &hash,
		&expiresAt, &createdAt, &revokedAt,
		&ipAddress, &userAgent, &wasRevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, domainerr.ErrRefreshTokenNotFound
		}
		return nil, nil, fmt.Errorf("revoke refresh token by hash: %w", err)
	}

	tokenID, err := valueobject.ParseRefreshTokenID(id)
	if err != nil {
		return nil, nil, fmt.Errorf("parse refresh token id: %w", err)
	}

	uid, err := valueobject.ParseUserID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse user id: %w", err)
	}

	token := entity.ReconstructRefreshToken(
		tokenID, uid, hash,
		expiresAt, createdAt, revokedAt,
		inetToString(ipAddress), derefString(userAgent),
	)

	return token, wasRevokedAt, nil
}

func (r *RefreshTokenRepositoryPg) RevokeByHashTx(ctx context.Context, tx repository.Transaction, tokenHash string, now time.Time) (*entity.RefreshToken, *time.Time, error) {
	const q = `
		WITH old_data AS (
			SELECT id, revoked_at FROM refresh_tokens WHERE token_hash = $1 FOR UPDATE
		),
		updated AS (
			UPDATE refresh_tokens
			SET revoked_at = COALESCE(refresh_tokens.revoked_at, $2)
			FROM old_data
			WHERE refresh_tokens.id = old_data.id
			RETURNING refresh_tokens.id, refresh_tokens.user_id, refresh_tokens.token_hash,
			          refresh_tokens.expires_at, refresh_tokens.created_at, refresh_tokens.revoked_at,
			          refresh_tokens.ip_address, refresh_tokens.user_agent, old_data.revoked_at AS was_revoked_at
		)
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at,
		       ip_address, user_agent, was_revoked_at FROM updated`

	var (
		id           string
		userID       string
		hash         string
		expiresAt    time.Time
		createdAt    time.Time
		revokedAt    *time.Time
		ipAddress     *netip.Prefix
		userAgent    *string
		wasRevokedAt *time.Time
	)

	pgxTx := unwrapTx(tx)
	err := pgxTx.QueryRow(ctx, q, tokenHash, now).Scan(
		&id, &userID, &hash,
		&expiresAt, &createdAt, &revokedAt,
		&ipAddress, &userAgent, &wasRevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, domainerr.ErrRefreshTokenNotFound
		}
		return nil, nil, fmt.Errorf("revoke refresh token by hash in transaction: %w", err)
	}

	tokenID, err := valueobject.ParseRefreshTokenID(id)
	if err != nil {
		return nil, nil, fmt.Errorf("parse refresh token id: %w", err)
	}

	uid, err := valueobject.ParseUserID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse user id: %w", err)
	}

	token := entity.ReconstructRefreshToken(
		tokenID, uid, hash,
		expiresAt, createdAt, revokedAt,
		inetToString(ipAddress), derefString(userAgent),
	)

	return token, wasRevokedAt, nil
}

func (r *RefreshTokenRepositoryPg) scanToken(ctx context.Context, query string, args ...any) (*entity.RefreshToken, error) {
	var (
		id        string
		userID    string
		tokenHash string
		expiresAt time.Time
		createdAt time.Time
		revokedAt *time.Time
		ipAddress  *netip.Prefix
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
		inetToString(ipAddress), derefString(userAgent),
	), nil
}
