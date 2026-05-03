package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// CreateShareLink generates a fresh share-link for a node. Owner-only.
// The raw token is returned in the response — and never again.
func (s *StorageService) CreateShareLink(ctx context.Context, actor *port.Actor, req dto.CreateShareRequest) (*dto.ShareResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.policy.allowOwner(actor, nil); err != nil {
		return nil, err
	}
	nodeID, err := valueobject.ParseNodeID(req.NodeID)
	if err != nil {
		return nil, err
	}
	node, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, nodeID)
	if err != nil {
		return nil, err
	}
	perm, err := valueobject.ParsePermission(req.Permission)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		d, parseErr := time.ParseDuration(req.ExpiresIn)
		if parseErr != nil || d <= 0 {
			return nil, dto.ErrInvalidExpiresIn
		}
		exp := now.Add(d)
		expiresAt = &exp
	}

	token, err := s.tokens.NewShareToken()
	if err != nil {
		return nil, fmt.Errorf("generate share token: %w", err)
	}
	shareID := s.ids.NewShareID()
	share, err := entity.NewShare(shareID, actor.UserID, node.ID(), token.Hash(), perm, expiresAt, now)
	if err != nil {
		return nil, err
	}
	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("persist share: %w", err)
	}

	resp := &dto.ShareResponse{
		ID:         share.ID().String(),
		NodeID:     share.NodeID().String(),
		Permission: share.Permission().String(),
		Token:      token.String(),
		URL:        joinURL(s.publicBaseURL, "/storage/v1/public/"+token.String()),
		ExpiresAt:  share.ExpiresAt(),
		RevokedAt:  share.RevokedAt(),
		CreatedAt:  share.CreatedAt(),
	}
	return resp, nil
}
