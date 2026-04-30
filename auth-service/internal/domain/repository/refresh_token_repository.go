package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// RefreshTokenRepository defines the interface for refresh token persistence.
// Implemented in infrastructure layer (Postgres).
// Returns domainerr sentinels on known failures.
type RefreshTokenRepository interface {
	// Save persists a new refresh token.
	Save(ctx context.Context, token *entity.RefreshToken) error

	// SaveTx persists a new refresh token within a transaction.
	SaveTx(ctx context.Context, tx pgx.Tx, token *entity.RefreshToken) error

	// FindByTokenHash retrieves a refresh token by its hash.
	// Returns domainerr.ErrRefreshTokenNotFound if not exists.
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)

	// RevokeByID marks a specific token as revoked.
	// Returns domainerr.ErrRefreshTokenNotFound if not exists.
	RevokeByID(ctx context.Context, id valueobject.RefreshTokenID, now time.Time) error

	// RevokeByHash atomically marks a token as revoked by its hash and returns it.
	// Returns the token state including whether it was already revoked.
	// Returns domainerr.ErrRefreshTokenNotFound if not exists.
	RevokeByHash(ctx context.Context, tokenHash string, now time.Time) (*entity.RefreshToken, *time.Time, error)

	// RevokeByHashTx atomically marks a token as revoked within a transaction.
	RevokeByHashTx(ctx context.Context, tx pgx.Tx, tokenHash string, now time.Time) (*entity.RefreshToken, *time.Time, error)

	// RevokeAllByUserID revokes all active tokens for a user (logout from all devices).
	RevokeAllByUserID(ctx context.Context, userID valueobject.UserID, now time.Time) error

	// DeleteExpired removes all tokens that expired before the given time.
	// Used for periodic cleanup to prevent table bloat.
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}
