package usecase

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// ListShareLinks lists shares attached to a node. Owner-only.
// Token values are NEVER returned (only token hash is stored anyway).
func (s *StorageService) ListShareLinks(ctx context.Context, actor *port.Actor, req dto.ListSharesRequest) (*dto.ListSharesResponse, error) {
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
	shares, err := s.shareRepo.ListByNode(ctx, actor.UserID, node.ID(), req.IncludeRevoked)
	if err != nil {
		return nil, err
	}
	resp := &dto.ListSharesResponse{Items: make([]dto.ShareResponse, 0, len(shares))}
	for _, sh := range shares {
		resp.Items = append(resp.Items, dto.ShareResponse{
			ID:         sh.ID().String(),
			NodeID:     sh.NodeID().String(),
			Permission: sh.Permission().String(),
			ExpiresAt:  sh.ExpiresAt(),
			RevokedAt:  sh.RevokedAt(),
			CreatedAt:  sh.CreatedAt(),
		})
	}
	return resp, nil
}
