package entity

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// RefreshToken represents a refresh token issued to a user.
// Has its own lifecycle: created -> active -> revoked/expired.
type RefreshToken struct {
	id        valueobject.RefreshTokenID
	userID    valueobject.UserID
	tokenHash string
	expiresAt time.Time
	createdAt time.Time
	revokedAt *time.Time
	ipAddress string
	userAgent string
}

// NewRefreshToken creates a new active refresh token.
func NewRefreshToken(
	id valueobject.RefreshTokenID,
	userID valueobject.UserID,
	tokenHash string,
	expiresAt time.Time,
	ipAddress string,
	userAgent string,
	now time.Time,
) *RefreshToken {
	return &RefreshToken{
		id:        id,
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
		createdAt: now,
		revokedAt: nil,
		ipAddress: ipAddress,
		userAgent: userAgent,
	}
}

// ReconstructRefreshToken reconstructs from persistence.
func ReconstructRefreshToken(
	id valueobject.RefreshTokenID,
	userID valueobject.UserID,
	tokenHash string,
	expiresAt time.Time,
	createdAt time.Time,
	revokedAt *time.Time,
	ipAddress string,
	userAgent string,
) *RefreshToken {
	return &RefreshToken{
		id:        id,
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
		createdAt: createdAt,
		revokedAt: revokedAt,
		ipAddress: ipAddress,
		userAgent: userAgent,
	}
}

// IsValid checks if token is active: not revoked and not expired.
// This is the core business rule for refresh tokens.
func (t *RefreshToken) IsValid(now time.Time) bool {
	return t.revokedAt == nil && now.Before(t.expiresAt)
}

// Revoke marks the token as revoked.
func (t *RefreshToken) Revoke(now time.Time) {
	t.revokedAt = &now
}

func (t *RefreshToken) ID() valueobject.RefreshTokenID { return t.id }
func (t *RefreshToken) UserID() valueobject.UserID     { return t.userID }
func (t *RefreshToken) TokenHash() string              { return t.tokenHash }
func (t *RefreshToken) ExpiresAt() time.Time           { return t.expiresAt }
func (t *RefreshToken) CreatedAt() time.Time           { return t.createdAt }
func (t *RefreshToken) RevokedAt() *time.Time          { return t.revokedAt }
func (t *RefreshToken) IPAddress() string              { return t.ipAddress }
func (t *RefreshToken) UserAgent() string              { return t.userAgent }
