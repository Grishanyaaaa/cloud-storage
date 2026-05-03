package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// RestoreNode clears deleted_at on the node and its subtree. Owner-only.
func (s *StorageService) RestoreNode(ctx context.Context, actor *port.Actor, nodeID string) (*dto.NodeResponse, error) {
	if err := s.policy.allowOwner(actor, nil); err != nil {
		return nil, err
	}
	id, err := valueobject.ParseNodeID(nodeID)
	if err != nil {
		return nil, err
	}
	node, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := node.Restore(now); err != nil {
		return nil, err
	}
	txErr := s.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		if _, err := s.nodeRepo.RestoreSubtreeTx(ctx, tx, node, now); err != nil {
			return fmt.Errorf("restore subtree: %w", err)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return toNodeResponse(node, nil), nil
}
