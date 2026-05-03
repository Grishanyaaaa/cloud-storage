package entity

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// Share is a public link granting `permission` to a node and its subtree.
// We store sha256(token) only; the raw token is shown to the user once
// at creation time and never returned again.
type Share struct {
	id         valueobject.ShareID
	nodeID     valueobject.NodeID
	ownerID    valueobject.UserID
	tokenHash  string
	permission valueobject.Permission
	expiresAt  *time.Time
	revokedAt  *time.Time
	createdAt  time.Time
}

// NewShare creates a fresh, alive share bound to (ownerID, nodeID).
// expiresAt is optional — pass nil for a non-expiring link.
// Caller is responsible for verifying that the node is owned by ownerID
// and not already deleted.
func NewShare(
	id valueobject.ShareID,
	owner valueobject.UserID,
	node valueobject.NodeID,
	tokenHash string,
	permission valueobject.Permission,
	expiresAt *time.Time,
	now time.Time,
) (*Share, error) {
	if !valueobject.IsValidTokenHash(tokenHash) {
		return nil, domainerr.ErrInvalidShareToken
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return nil, domainerr.ErrInvalidExpiry
	}
	return &Share{
		id:         id,
		nodeID:     node,
		ownerID:    owner,
		tokenHash:  tokenHash,
		permission: permission,
		expiresAt:  expiresAt,
		revokedAt:  nil,
		createdAt:  now,
	}, nil
}

// ReconstructShare reconstructs a Share from persistence.
func ReconstructShare(
	id valueobject.ShareID,
	nodeID valueobject.NodeID,
	ownerID valueobject.UserID,
	tokenHash string,
	permission valueobject.Permission,
	expiresAt *time.Time,
	revokedAt *time.Time,
	createdAt time.Time,
) *Share {
	return &Share{
		id:         id,
		nodeID:     nodeID,
		ownerID:    ownerID,
		tokenHash:  tokenHash,
		permission: permission,
		expiresAt:  expiresAt,
		revokedAt:  revokedAt,
		createdAt:  createdAt,
	}
}

// Getters
func (s *Share) ID() valueobject.ShareID            { return s.id }
func (s *Share) NodeID() valueobject.NodeID         { return s.nodeID }
func (s *Share) OwnerID() valueobject.UserID        { return s.ownerID }
func (s *Share) TokenHash() string                  { return s.tokenHash }
func (s *Share) Permission() valueobject.Permission { return s.permission }
func (s *Share) ExpiresAt() *time.Time              { return s.expiresAt }
func (s *Share) RevokedAt() *time.Time              { return s.revokedAt }
func (s *Share) CreatedAt() time.Time               { return s.createdAt }

// IsRevoked returns true when the share was explicitly revoked.
func (s *Share) IsRevoked() bool { return s.revokedAt != nil }

// IsExpired returns true when expiresAt is set and ≤ now.
func (s *Share) IsExpired(now time.Time) bool {
	return s.expiresAt != nil && !now.Before(*s.expiresAt)
}

// IsActive returns true when not revoked and not expired.
func (s *Share) IsActive(now time.Time) bool {
	return !s.IsRevoked() && !s.IsExpired(now)
}

// Revoke marks the share as revoked (idempotent).
func (s *Share) Revoke(now time.Time) {
	if s.revokedAt == nil {
		s.revokedAt = &now
	}
}

// AssertActive returns a typed domain error if the share is not active.
func (s *Share) AssertActive(now time.Time) error {
	if s.IsRevoked() {
		return domainerr.ErrShareRevoked
	}
	if s.IsExpired(now) {
		return domainerr.ErrShareExpired
	}
	return nil
}

// AssertCovers verifies that target's path is inside (or equal to) the share root path.
// shareRoot must be the Node referenced by s.nodeID; passing any other node is a programming error.
func (s *Share) AssertCovers(shareRoot, target *Node) error {
	if !shareRoot.ID().Equals(s.nodeID) {
		return domainerr.ErrShareScopeViolation
	}
	if !shareRoot.Path().CoversOrEquals(target.Path()) {
		return domainerr.ErrShareScopeViolation
	}
	return nil
}
