package usecase

import (
	"context"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// ResolveShareToken validates a raw share token and assembles an Actor for downstream use cases.
//
// Order of checks (см. blueprint §13.6):
//  1. Format validation.
//  2. Look up by sha256(token) — returns ErrShareNotFound if absent.
//  3. Reject revoked / expired shares.
//  4. Load the share root node (must be alive).
func (s *StorageService) ResolveShareToken(ctx context.Context, rawToken string) (*port.Actor, error) {
	token, err := valueobject.ParseShareToken(rawToken)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	share, err := s.shareRepo.GetActiveByTokenHash(ctx, token.Hash(), now)
	if err != nil {
		return nil, err
	}
	if err := share.AssertActive(now); err != nil {
		return nil, err
	}
	root, err := s.nodeRepo.GetByID(ctx, share.NodeID())
	if err != nil {
		return nil, err
	}
	if root.IsDeleted() {
		return nil, domainerr.ErrShareScopeViolation
	}
	return &port.Actor{
		Kind:      port.ActorKindShareLink,
		UserID:    share.OwnerID(),
		Roles:     nil,
		Share:     share,
		ShareRoot: root,
	}, nil
}
