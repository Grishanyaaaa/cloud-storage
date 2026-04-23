package repository

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// RefreshTokenRepository defines the interface for refresh token persistence.
// Implemented in infrastructure layer (Postgres).
// Returns domainerr sentinels on known failures.
type RefreshTokenRepository interface {
	// Save persists a new refresh token.
	Save(ctx context.Context, token *entity.RefreshToken) error

	// FindByTokenHash retrieves a refresh token by its hash.
	// Returns domainerr.ErrRefreshTokenNotFound if not exists.
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)

	// RevokeByID marks a specific token as revoked.
	// Returns domainerr.ErrRefreshTokenNotFound if not exists.
	RevokeByID(ctx context.Context, id valueobject.RefreshTokenID) error

	// RevokeAllByUserID revokes all active tokens for a user (logout from all devices).
	RevokeAllByUserID(ctx context.Context, userID valueobject.UserID) error
}
